package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pborenstein/gh-repopotamus/internal/move"
	"github.com/pborenstein/gh-repopotamus/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Browse two owners side by side and move repos between them",
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
	res, err := tui.Run()
	if err != nil {
		return err
	}
	if !res.Ran {
		return nil
	}

	// The TUI ran in the alternate screen, which is gone now. Leave a record
	// in the scrollback.
	var moved, failed, skipped int
	for _, p := range res.Plans {
		fmt.Printf("%s -> %s\n", p.Src, p.Dst)
		for _, it := range p.Items {
			line := fmt.Sprintf("  %-8s %s", it.Status, it.Name)
			if it.Reason != "" {
				line += "  (" + it.Reason + ")"
			}
			if it.Err != nil {
				line += "  " + it.Err.Error()
			}
			fmt.Println(line)
		}
		moved += p.Count(move.Moved)
		failed += p.Count(move.Failed)
		skipped += p.Count(move.Skipped)
	}
	fmt.Printf("\n%d moved, %d failed, %d skipped.\n", moved, failed, skipped)
	if failed > 0 {
		return fmt.Errorf("%d transfer(s) failed", failed)
	}
	return nil
}
