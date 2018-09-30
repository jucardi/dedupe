package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/jucardi/dedupe/cmd/dedupe/version"
	"github.com/jucardi/dedupe/dedupe"
	"github.com/jucardi/go-logger-lib/log"
	a "github.com/logrusorgru/aurora"
	"os"
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
	printReport(result)
}

func validate(args []string) bool {
	return len(args) == 1
}

func printReport(report *dedupe.DupeReport) {
	fmt.Println()
	fmt.Println(a.Bold(a.Blue("Duplicates:")))

	for k, v := range report.Dupes {
		fmt.Println()
		fmt.Println(a.Green("Checksum: "), a.Cyan(k))

		for _, f := range v {
			fmt.Println(a.Gray("- " + f))
		}
	}

	fmt.Println()

	if len(report.Errors) > 0 {
		fmt.Println(a.Bold(a.Red("Errors:")))
		for _, e := range report.Errors {
			fmt.Println(a.Gray("- " + e.Error()))
		}
	} else {
		fmt.Println(a.Green("No errors."))
		fmt.Println()
	}

}
