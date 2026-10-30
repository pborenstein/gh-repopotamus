package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/pborenstein/gh-repopotamus/internal/github"
	"github.com/pborenstein/gh-repopotamus/internal/selection"
	"github.com/spf13/cobra"
)

var (
	listLong    bool
	listFilters filterFlags
)

var listCmd = &cobra.Command{
	Use:   "list OWNER",
	Short: "List repos owned by an org or user",
	Long: `List repos owned by an org or user, narrowed by the same filters as move.
The plain output is one name per line, ready to pipe into move.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := listFilters.criteria()
		if err != nil {
			return err
		}
		all, err := github.ListRepos(args[0])
		if err != nil {
			return err
		}
		repos, _, err := selection.Select(all, c)
		if err != nil {
			return err
		}

		if !listLong {
			for _, r := range repos {
				fmt.Println(r.Name)
			}
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tVISIBILITY\tARCHIVED\tFORK\tLANGUAGE\tPUSHED\tTOPICS")
		for _, r := range repos {
			fmt.Fprintf(w, "%s\t%s\t%t\t%t\t%s\t%s\t%s\n",
				r.Name,
				strings.ToLower(r.Visibility),
				r.Archived,
				r.Fork,
				r.Language,
				r.PushedAt.Format("2006-01-02"),
				strings.Join(r.Topics, ","),
			)
		}
		return w.Flush()
	},
}

func init() {
	listFilters.register(listCmd.Flags())
	listCmd.Flags().BoolVarP(&listLong, "long", "l", false, "show metadata columns")
	rootCmd.AddCommand(listCmd)
}
