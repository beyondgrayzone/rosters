package main

import (
	"os"
	"time"

	"rosters/cmd/skl/commands"
	"rosters/pkg/format"

	"github.com/spf13/cobra"
)

const VERSION = "0.1.0"

var (
	quiet   bool
	jsonOut bool
	verbose bool
	timing  bool
)

var rootCmd = &cobra.Command{
	Use:   "rt",
	Short: "Git Native Issue Tracker for AI Agents",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	commands.RegisterInitCommand(rootCmd)
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output as structured JSON")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Show full details in output")
	rootCmd.PersistentFlags().BoolVar(&timing, "timing", false, "Print execution time to stderr")
	rootCmd.Version = VERSION
}

func main() {
	startTime := time.Now()
	format.SetQuiet(quiet)
	format.SetJSONMode(jsonOut)

	if err := rootCmd.Execute(); err != nil {
		format.PrintError(err.Error())
		os.Exit(1)
	}
	if timing {
		format.PrintTiming(time.Since(startTime))
	}
}
