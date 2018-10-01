package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/jucardi/dedupe/cmd/dedupe/version"
	"github.com/jucardi/dedupe/dedupe"
	"github.com/jucardi/go-logger-lib/log"
	"os"
	"github.com/jucardi/dedupe/shutdown"
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
	rootCmd.Flags().StringP("save-to", "s", "", "If provided, it will save the progress of the report and their resolution")
	rootCmd.Flags().StringP("load-from", "l", "", "If provided, loads a previous progress to continue promting for resolution of duplicates")
	rootCmd.Flags().StringP("algorithm", "a", string(dedupe.HashSHA256), "Indicates the hashing algorithm to use for checksums (md5, sha256). Default is sha256")
	rootCmd.Flags().BoolP("recursive", "r", false, "Indicates if dedupe should find dupes recursively. Default is false")
	rootCmd.Flags().BoolP("keep-one", "o", false, "Enables the 'keep one' mode. At the end of the report, for each duplication it dedupe will ask which file to keep")
	rootCmd.Flags().BoolP("dry-run", "d", false, "Combined with 'keep-one', it prints the files that will be deleted without taking any actions")
	rootCmd.Flags().BoolP("verbose", "v", false, "Enables verbose mode")
	rootCmd.Execute()
}

func printUsage(cmd *cobra.Command) {
	cmd.Println(fmt.Sprintf(long, version.Version, version.Built))
	cmd.Usage()
}

func initCmd(cmd *cobra.Command, args []string) {
	cmd.Use = fmt.Sprintf(usage, cmd.Use)
}

func run(cmd *cobra.Command, args []string) {
	shutdown.ListenForSignals()
	log.Info("Listening for shutdown signals")

	recursive, _ := cmd.Flags().GetBool("recursive")
	algorithm, _ := cmd.Flags().GetString("algorithm")
	keepone, _ := cmd.Flags().GetBool("keep-one")
	dryrun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")
	load, _ := cmd.Flags().GetString("load-from")
	save, _ := cmd.Flags().GetString("save-to")

	if !validate(args) && load == "" {
		log.Error("No starting path or progress file provided")
		printUsage(cmd)
		os.Exit(-1)
	}

	c := &cli{
		Verbose:   verbose,
		Recursive: recursive,
		Algorithm: dedupe.HashMode(algorithm),
		KeepOne:   keepone,
		DryRun:    dryrun,
		SaveTo:    save,
	}

	if load != "" {
		if save == "" {
			c.SaveTo = load
		}
		c.Load(load)
	} else {
		c.Start(args[0])
	}
}

func validate(args []string) bool {
	return len(args) == 1
}
