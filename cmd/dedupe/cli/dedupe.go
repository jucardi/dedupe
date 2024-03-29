package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jucardi/dedupe/dedupe"
	"github.com/jucardi/dedupe/shutdown"
	"github.com/jucardi/go-logger-lib/log"
	"github.com/jucardi/go-strings/stringx"
	a "github.com/logrusorgru/aurora"
)

type cli struct {
	Algorithm dedupe.HashMode
	Recursive bool
	Verbose   bool
	KeepOne   bool
	DryRun    bool
	SaveTo    string
}

func (c *cli) Start(path string) {
	opts := &dedupe.Options{
		Recursive: c.Recursive,
		Mode:      c.Algorithm,
	}

	if c.Verbose {
		opts.CurrentDirCallback = printWorkingDirectory
		opts.ReadingHashCallback = printCalculatingHash
		opts.HashReadCallback = printFileHash
	}

	instance := dedupe.New()
	instance.SetOptions(opts)

	result, err := instance.FindDupes(path)

	if err != nil {
		log.Errorf("Unable to find duplicates. %s", err.Error())
		os.Exit(1)
	}

	c.handleReport(result)
}

func (c *cli) Load(file string) {
	fmt.Println(a.Green("Loading report from "), a.Cyan(file))
	fmt.Println(a.Green("Please wait . . ."))

	data, err := os.ReadFile(file)

	if err != nil {
		log.Panic("Unable to read file", err.Error())
	}

	var report *dedupe.DupeReport

	if err := json.Unmarshal(data, report); err != nil {
		log.Panic("Unable to unmarshal report file", err.Error())
	}

	c.handleReport(report)
}

func (c *cli) handleReport(report *dedupe.DupeReport) {
	remaining := &dedupe.DupeReport{
		Errors: report.Errors,
		Dupes:  map[string][]string{},
	}

	if c.SaveTo != "" {
		log.Info("Shutdown hook registered.")
		shutdown.AddShutdownHook(func() error {
			log.Info("exiting")
			return saveProgress(c.SaveTo, remaining)
		})

		for k, v := range report.Dupes {
			remaining.Dupes[k] = v
		}
	}

	if len(report.Errors) > 0 {
		fmt.Println(a.Bold(a.Red("Errors:")))
		for _, e := range report.Errors {
			fmt.Println(a.Gray(16, "- "+e.Error()))
		}
	} else {
		fmt.Println()
		fmt.Println(a.Green("No errors."))
		fmt.Println()
	}

	if len(report.Dupes) == 0 {
		fmt.Println(a.Green("No duplicates."))
		fmt.Println()
		return
	}

	fmt.Println()
	fmt.Println(a.Bold(a.Blue("Duplicates:")))

	count := 0
	for k, v := range report.Dupes {
		fmt.Println()
		fmt.Println(a.Gray(12, fmt.Sprint("Items left:")), a.Gray(20, fmt.Sprint(len(report.Dupes)-count)))
		fmt.Println(a.Green("Checksum:  "), a.Cyan(k))

		count++
		if c.KeepOne {
			c.askSingleChoice(v)
			continue
		}

		for _, f := range v {
			fmt.Println(a.Gray(12, "- "+f))
		}

		if c.SaveTo != "" {
			delete(remaining.Dupes, k)
		}
	}

	fmt.Println()
}

func (c *cli) askSingleChoice(files []string) {
	var err = errors.New("initial")
	fmt.Println(a.Green("Which file would you like to keep?"))

	for err != nil {
		fmt.Println()
		c.printCommonChoice("a", "  All")
		c.printCommonChoice("n", "  None")
		c.printCommonChoice("s {}", "Keeps {} and replaces the other files with symbolic links to {}", "N")
		fmt.Println()

		for i, f := range files {
			c.printChoice(strconv.Itoa(i+1), f)
		}

		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		choice := stringx.New(text).ToLower().Trim("\n").S()

		switch choice {
		case "n":
			c.deleteFiles(files)
			fallthrough
		case "a":
			err = nil
		default:
			var (
				val     string
				symlink bool
			)

			if strings.HasPrefix(choice, "s") {
				val = stringx.New(choice).Replace("s", "", 1).TrimSpace().S()
				symlink = true
			} else {
				val = choice
			}

			if j, e := strconv.ParseInt(val, 0, 0); e != nil || int(j) <= 0 || int(j) > len(files) {
				fmt.Println(a.Red("Invalid choice"))
			} else {
				deletedFiles := append(files[:j-1], files[j:]...)
				c.deleteFiles(deletedFiles)
				err = nil

				if symlink {
					src, e := filepath.Abs(files[j-1])
					if e != nil {
						fmt.Println(a.Red(e.Error()))
						continue
					}
					for _, oldFile := range deletedFiles {
						trg, e := filepath.Abs(oldFile)
						if e != nil {
							fmt.Println(a.Red(e.Error()))
							continue
						}
						if c.DryRun {
							fmt.Println(a.Bold(a.Green("(symlink to be created) ")), src, " > ", trg)
						} else if e = os.Symlink(src, trg); e != nil {
							fmt.Println(a.Red(e.Error()))
						} else {
							fmt.Println(a.Bold(a.Green("(symlink) ")), oldFile, " > ", files[j])
						}
					}
				}
			}
		}
	}
}

func (c *cli) printChoice(s string, option string) {
	fmt.Printf("  (%s) %s", a.Bold(a.Yellow(s)), a.Bold(a.Blue(option)))
	fmt.Println()
}

func (c *cli) printCommonChoice(s string, option string, indicator ...string) {
	if len(indicator) > 0 {
		sx := strings.Replace(s, "{}", fmt.Sprint(a.Gray(12, indicator[0])), -1)
		ox := strings.Replace(option, "{}", fmt.Sprint(a.Gray(12, indicator[0])), -1)
		fmt.Printf("  (%s) %s", a.Bold(a.Yellow(sx)), a.Bold(a.Cyan(ox)))
	} else {
		fmt.Printf("  (%s) %s", a.Bold(a.Yellow(s)), a.Bold(a.Cyan(option)))
	}

	fmt.Println()
}

func (c *cli) deleteFiles(files []string) {
	for _, v := range files {
		if c.DryRun {
			fmt.Println(a.Bold(a.Magenta("(to delete) ")), v)
			continue
		}

		if err := os.Remove(v); err != nil {
			fmt.Println(a.Red("Unable to delete file "), v)
			fmt.Println(a.Red("    "), err.Error())
		} else {
			fmt.Println(a.Bold(a.Red("(deleted) ")), v)
		}
	}
}

func printWorkingDirectory(dir string) {
	fmt.Println(a.Green("  > Checking contents in directory: "), a.Cyan(dir))
}

func printCalculatingHash(file string) {
	fmt.Print(a.Green("  > Calculating Hash on file: "), a.Cyan(file))
}

func printFileHash(file, hash string) {
	fmt.Println("			> ", a.Bold(a.Red(hash)))
}

func saveProgress(target string, report *dedupe.DupeReport) error {
	fmt.Println(a.Green("Saving report to "), a.Cyan(target))
	fmt.Println(a.Green("Please wait . . ."))

	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("error marshalling report, %s", err.Error())
	}
	file, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("unable to create report file, %s", err.Error())
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("unable to write data to report file, %s", err.Error())
	}
	return file.Sync()
}
