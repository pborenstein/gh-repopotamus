package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pborenstein/gh-repopotamus/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Pick owners and repos interactively",
	Args:  cobra.NoArgs,
	RunE:  runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
	// With no subcommand, gh repopotamus starts the TUI.
	rootCmd.Args = cobra.NoArgs
	rootCmd.RunE = runTUI
}

func runTUI(cmd *cobra.Command, args []string) error {
	src, dst, ok, err := tui.PickOwners()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	// The repo list comes next; for now, report the choice.
	fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s\n", src, dst)
	return nil
}
