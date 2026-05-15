package main

import (
	"os"
	"time"

	"rosters/cmd/rt/commands"
	"rosters/pkg/format"

	"github.com/spf13/cobra"
)

const VERSION = "0.1.0"

var (
	quiet      bool
	jsonOut    bool
	verbose    bool
	timing     bool
	formatFlag string
)

var rootCmd = &cobra.Command{
	Use:     "rt",
	Short:   "Git Native Issue Tracker for AI Agents",
	Version: VERSION,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	commands.RegisterInitCommand(rootCmd)
	commands.RegisterDoctorCommand(rootCmd)
	commands.RegisterCreateCommand(rootCmd)
	commands.RegisterShowCommand(rootCmd)
	commands.RegisterListCommand(rootCmd)
	commands.RegisterSearchCommand(rootCmd)
	commands.RegisterUpdateCommand(rootCmd)
	commands.RegisterCloseCommand(rootCmd)
	commands.RegisterDepCommand(rootCmd)
	commands.RegisterBlockCommand(rootCmd)
	commands.RegisterUnblockCommand(rootCmd)
	commands.RegisterBlockedCommand(rootCmd)
	commands.RegisterReadyCommand(rootCmd)
	commands.RegisterLabelCommand(rootCmd)
	commands.RegisterPlanCommand(rootCmd)
	commands.RegisterTplCommand(rootCmd)

	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output as structured JSON")
	rootCmd.PersistentFlags().StringVar(&formatFlag, "format", "markdown", "Output format (markdown, compact, plain, ids, json)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Show full details in output")
	rootCmd.PersistentFlags().BoolVar(&timing, "timing", false, "Print execution time to stderr")
}

func main() {
	startTime := time.Now()

	if err := rootCmd.ParseFlags(os.Args); err == nil {
		format.SetQuiet(quiet)
		if jsonOut {
			format.SetFormat("json")
		} else {
			format.SetFormat(formatFlag)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		format.PrintError(err.Error())
		os.Exit(1)
	}

	if timing {
		format.PrintTiming(time.Since(startTime))
	}
}
