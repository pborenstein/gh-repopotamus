package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/pborenstein/gh-repopotamus/internal/clone"
	"github.com/pborenstein/gh-repopotamus/internal/github"
	"github.com/pborenstein/gh-repopotamus/internal/selection"
	"github.com/spf13/cobra"
)

var (
	cloneFilters  filterFlags
	cloneFromFile string
	cloneDir      string
	cloneYes      bool
	cloneDryRun   bool
)

var cloneCmd = &cobra.Command{
	Use:   "clone OWNER [REPO...]",
	Short: "Clone an owner's repos into a local directory",
	Long: `Clone repos from OWNER into DIR, one folder per repo (DIR/REPO).

Repos are chosen the same way as for move: by name (positional, '-' for
stdin, or --from-file), then narrowed by --match, --regex and the metadata
filters. With no names and no filters, every repo in OWNER is cloned.
Repos whose folder already exists are skipped.`,
	Example: `  gh repopotamus clone pborenstein site-blog site-docs
  gh repopotamus clone my-org --dir ~/src/my-org --no-archived
  gh repopotamus list pborenstein --topic go | gh repopotamus clone pborenstein - --dir ~/src`,
	Args: cobra.MinimumNArgs(1),
	RunE: runClone,
}

func init() {
	cloneFilters.register(cloneCmd.Flags())
	cloneCmd.Flags().StringVar(&cloneFromFile, "from-file", "", "read repo names from PATH, one per line")
	cloneCmd.Flags().StringVar(&cloneDir, "dir", "", "clone into DIR (default: the current directory)")
	cloneCmd.Flags().BoolVarP(&cloneYes, "yes", "y", false, "skip the confirmation prompt")
	cloneCmd.Flags().BoolVar(&cloneDryRun, "dry-run", false, "print the plan and exit")
	rootCmd.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) error {
	owner := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir, err := clone.ResolveDir(cloneDir, cwd)
	if err != nil {
		return err
	}

	c, err := cloneFilters.criteria()
	if err != nil {
		return err
	}
	names, usedStdin, err := gatherNames(owner, args[1:], cloneFromFile)
	if err != nil {
		return err
	}
	c.Names = names

	repos, err := github.ListRepos(owner)
	if err != nil {
		return err
	}
	selected, missing, err := selection.Select(repos, c)
	if err != nil {
		return err
	}

	plan := clone.NewPlan(dir, selected, clone.Exists)
	plan.AddMissing(owner, missing)
	if err := printClonePlan(plan); err != nil {
		return err
	}

	pending := plan.Count(clone.Pending)
	if pending == 0 {
		fmt.Println("\nNothing to clone.")
		return nil
	}
	if cloneDryRun {
		fmt.Printf("\nDry run: %d to clone into %s, %d skipped.\n", pending, dir, plan.Count(clone.Skipped))
		return nil
	}
	if !cloneYes {
		ok, err := confirm(fmt.Sprintf("\nClone %d repos into %s? [y/N] ", pending, dir), usedStdin)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Aborted.")
			return nil
		}
	}

	fmt.Println()
	clone.Execute(&plan, github.Clone, func(it clone.Item) {
		if it.Status == clone.Failed {
			fmt.Printf("  failed  %s: %v\n", it.FullName(), it.Err)
		} else {
			fmt.Printf("  cloned  %s\n", it.FullName())
		}
	})

	failed := plan.Count(clone.Failed)
	fmt.Printf("\n%d cloned, %d failed, %d skipped.\n", plan.Count(clone.Cloned), failed, plan.Count(clone.Skipped))
	if failed > 0 {
		return fmt.Errorf("%d clones failed", failed)
	}
	return nil
}

func printClonePlan(p clone.Plan) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "REPO\tPATH\tACTION")
	for _, it := range p.Items {
		action := "clone"
		if it.Status == clone.Skipped {
			action = "skip: " + it.Reason
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", it.FullName(), p.Path(it), action)
	}
	return w.Flush()
}
