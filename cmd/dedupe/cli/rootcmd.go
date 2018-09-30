package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/jucardi/dedupe/cmd/dedupe/version"
)

const (
	usage = ``
	long  = ``
)

var rootCmd = &cobra.Command{
	Use:              "infuse",
	Short:            "Parses a Golang template",
	Long:             fmt.Sprintf(long, version.Version, version.Built),
	PersistentPreRun: initCmd,
	Run:              run,
}

// Execute starts the execution of the run command.
func Execute() {
	rootCmd.Flags().StringP("someflag", "s", "", "someflag")
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
	someflag, _ := cmd.Flags().GetString("someflag")
	println(someflag)
}
