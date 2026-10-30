package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/pborenstein/gh-repopotamus/internal/github"
	"github.com/pborenstein/gh-repopotamus/internal/move"
	"github.com/pborenstein/gh-repopotamus/internal/selection"
	"github.com/spf13/cobra"
)

var (
	moveFilters  filterFlags
	moveFromFile string
	moveAll      bool
	moveYes      bool
	moveDryRun   bool
)

var moveCmd = &cobra.Command{
	Use:   "move SRC DST [REPO...]",
	Short: "Transfer repos from one owner to another",
	Long: `Transfer repos from SRC to DST.

Repos are chosen by name (positional, '-' for stdin, or --from-file), then
narrowed by --match, --regex and the metadata filters. With no names, the
filters apply to every repo in SRC. Moving every repo needs --all.`,
	Example: `  gh repopotamus move pborenstein my-org site-blog site-docs
  gh repopotamus move pborenstein my-org --match 'site-*' --no-archived
  gh repopotamus list pborenstein --pushed-before 2020-01-01 | gh repopotamus move pborenstein attic -`,
	Args: cobra.MinimumNArgs(2),
	RunE: runMove,
}

func init() {
	moveFilters.register(moveCmd.Flags())
	moveCmd.Flags().StringVar(&moveFromFile, "from-file", "", "read repo names from PATH, one per line")
	moveCmd.Flags().BoolVar(&moveAll, "all", false, "allow selecting every repo in SRC")
	moveCmd.Flags().BoolVarP(&moveYes, "yes", "y", false, "skip the confirmation prompt")
	moveCmd.Flags().BoolVar(&moveDryRun, "dry-run", false, "print the plan and exit")
	rootCmd.AddCommand(moveCmd)
}

func runMove(cmd *cobra.Command, args []string) error {
	src, dst := args[0], args[1]
	if strings.EqualFold(src, dst) {
		return fmt.Errorf("SRC and DST are both %s", src)
	}

	c, err := moveFilters.criteria()
	if err != nil {
		return err
	}
	names, usedStdin, err := gatherNames(src, args[2:], moveFromFile)
	if err != nil {
		return err
	}
	c.Names = names
	if len(names) == 0 && !c.Narrows() && !moveAll {
		return fmt.Errorf("no names or filters given; pass --all to move every repo in %s", src)
	}

	srcRepos, err := github.ListRepos(src)
	if err != nil {
		return err
	}
	selected, missing, err := selection.Select(srcRepos, c)
	if err != nil {
		return err
	}
	dstRepos, err := github.ListRepos(dst)
	if err != nil {
		return err
	}

	plan := move.NewPlan(src, dst, selected, missing, dstRepos)
	if err := printPlan(plan); err != nil {
		return err
	}

	pending := plan.Count(move.Pending)
	if pending == 0 {
		fmt.Println("\nNothing to move.")
		return nil
	}
	if moveDryRun {
		fmt.Printf("\nDry run: %d to move, %d skipped.\n", pending, plan.Count(move.Skipped))
		return nil
	}
	if !moveYes {
		ok, err := confirm(fmt.Sprintf("\nMove %d repos from %s to %s? [y/N] ", pending, src, dst), usedStdin)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Aborted.")
			return nil
		}
	}

	fmt.Println()
	move.Execute(&plan, github.Transfer, func(it move.Item) {
		if it.Status == move.Failed {
			fmt.Printf("  failed  %s: %v\n", it.Name, it.Err)
		} else {
			fmt.Printf("  moved   %s\n", it.Name)
		}
	})

	failed := plan.Count(move.Failed)
	fmt.Printf("\n%d moved, %d failed, %d skipped.\n", plan.Count(move.Moved), failed, plan.Count(move.Skipped))
	if failed > 0 {
		return fmt.Errorf("%d transfers failed", failed)
	}
	return nil
}

// gatherNames collects names from positional args ('-' reads stdin) and
// --from-file. It reports whether stdin was consumed, so the confirmation
// prompt knows to read from the terminal instead.
func gatherNames(owner string, args []string, fromFile string) (names []string, usedStdin bool, err error) {
	for _, a := range args {
		if a == "-" {
			if usedStdin {
				continue
			}
			usedStdin = true
			more, err := selection.ReadNames(os.Stdin, owner)
			if err != nil {
				return nil, false, fmt.Errorf("reading stdin: %w", err)
			}
			names = append(names, more...)
			continue
		}
		n, err := selection.StripOwner(a, owner)
		if err != nil {
			return nil, false, err
		}
		names = append(names, n)
	}

	if fromFile != "" {
		f, err := os.Open(fromFile)
		if err != nil {
			return nil, false, err
		}
		defer f.Close()
		more, err := selection.ReadNames(f, owner)
		if err != nil {
			return nil, false, fmt.Errorf("reading %s: %w", fromFile, err)
		}
		names = append(names, more...)
	}
	return names, usedStdin, nil
}

func printPlan(p move.Plan) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "REPO\tFROM\tTO\tACTION")
	for _, it := range p.Items {
		action := "move"
		if it.Status == move.Skipped {
			action = "skip: " + it.Reason
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", it.Name, p.Src, p.Dst, action)
	}
	return w.Flush()
}

// confirm asks a y/N question. If stdin held the repo names, the answer comes
// from the terminal instead.
func confirm(prompt string, usedStdin bool) (bool, error) {
	var in io.Reader = os.Stdin
	if usedStdin {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			return false, fmt.Errorf("names came from stdin and there is no terminal to confirm on; pass --yes")
		}
		defer tty.Close()
		in = tty
	}

	fmt.Print(prompt)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
