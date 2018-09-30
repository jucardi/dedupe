package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/jucardi/dedupe/cmd/dedupe/version"
	"github.com/jucardi/dedupe/dedupe"
	"github.com/jucardi/go-logger-lib/log"
	a "github.com/logrusorgru/aurora"
	"os"
	"errors"
	"bufio"
	"strconv"
	"strings"
	"github.com/jucardi/go-strings/stringx"
)

const (
	usage = `%s -r -a md5 .`
	long  = `
Dedupe - Duplicates finder
    Version: V-%s
    Built: %s
`
)

var rootCmd = &cobra.Command{
	Use:              "dedupe",
	Short:            "finds duplicates in the given path",
	Long:             fmt.Sprintf(long, version.Version, version.Built),
	PersistentPreRun: initCmd,
	Run:              run,
}

// Execute starts the execution of the run command.
func Execute() {
	rootCmd.Flags().BoolP("recursive", "r", false, "Indicates if dedupe should find dupes recursively. Default is false")
	rootCmd.Flags().StringP("algorithm", "a", string(dedupe.HashSHA256), "Indicates the hashing algorithm to use for checksums (md5, sha256). Default is sha256")
	rootCmd.Flags().BoolP("keep-one", "o", false, "Enables the 'keep one' mode. At the end of the report, for each duplication it dedupe will ask which file to keep")
	rootCmd.Flags().BoolP("dry-run", "d", false, "Combined with 'keep-one', it prints the files that will be deleted without taking any actions.")
	rootCmd.Execute()
}

func printUsage(cmd *cobra.Command) {
	cmd.Println(fmt.Sprintf(long, version.Version, version.Built))
	cmd.Usage()
}

func initCmd(cmd *cobra.Command, args []string) {
	// FromCommand(cmd) // if using a configurator
	cmd.Use = fmt.Sprintf(usage, cmd.Use)
}

func run(cmd *cobra.Command, args []string) {
	if !validate(args) {
		log.Error("Unexpected number of arguments")
		printUsage(cmd)
		os.Exit(-1)
	}

	recursive, _ := cmd.Flags().GetBool("recursive")
	algorithm, _ := cmd.Flags().GetString("algorithm")
	keepone, _ := cmd.Flags().GetBool("keep-one")
	dryrun, _ := cmd.Flags().GetBool("dry-run")

	instance := dedupe.New()
	instance.SetOptions(&dedupe.Options{
		Recursive: recursive,
		Mode:      dedupe.HashMode(algorithm),
	})

	result, err := instance.FindDupes(args[0])

	if err != nil {
		log.Errorf("Unable to find duplicates. %s", err.Error())
		os.Exit(1)
	}
	printReport(result, keepone, dryrun)
}

func validate(args []string) bool {
	return len(args) == 1
}

func printReport(report *dedupe.DupeReport, keepone, dryrun bool) {
	fmt.Println()
	fmt.Println(a.Bold(a.Blue("Duplicates:")))

	if len(report.Errors) > 0 {
		fmt.Println(a.Bold(a.Red("Errors:")))
		for _, e := range report.Errors {
			fmt.Println(a.Gray("- " + e.Error()))
		}
	} else {
		fmt.Println(a.Green("No errors."))
		fmt.Println()
	}

	for k, v := range report.Dupes {
		fmt.Println()
		fmt.Println(a.Green("Checksum: "), a.Cyan(k))

		if keepone {
			askSingleChoice(k, v, dryrun)
			continue
		}

		for _, f := range v {
			fmt.Println(a.Gray("- " + f))
		}
	}

	fmt.Println()
}

func askSingleChoice(checksum string, files []string, dryrun bool) {
	var err = errors.New("initial")
	fmt.Println(a.Blue("Which file would you like to keep?"))

	for err != nil {
		printChoice("a", "All")
		printChoice("n", "None")

		for i, f := range files {
			printChoice(strconv.Itoa(i+1), f)
		}

		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		choice := stringx.New(text).ToLower().Trim("\n").S()

		switch choice {
		case "n":
			deleteFiles(files, dryrun)
			fallthrough
		case "a":
			err = nil
		default:
			if j, e := strconv.ParseInt(strings.Trim(text, "\n"), 0, 0); e != nil || int(j) <= 0 || int(j) > len(files) {
				fmt.Println(a.Red("Invalid choice"))
			} else {
				deleteFiles(append(files[:j-1], files[j:]...), dryrun)
				err = nil
			}
		}
	}
}

func printChoice(s string, option string) {
	fmt.Printf("  (%s) %s", a.Bold(a.Brown(s)), a.Bold(a.Blue(option)))
	fmt.Println()
}

func deleteFiles(files []string, dryrun bool) {
	println("deleting")
	for _, v := range files {
		if dryrun {
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
