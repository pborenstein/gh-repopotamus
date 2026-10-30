package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/pborenstein/gh-repopotamus/internal/github"
	"github.com/spf13/cobra"
)

var listLong bool

var listCmd = &cobra.Command{
	Use:   "list OWNER",
	Short: "List repos owned by an org or user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := github.ListRepos(args[0])
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
	listCmd.Flags().BoolVarP(&listLong, "long", "l", false, "show metadata columns")
	rootCmd.AddCommand(listCmd)
}
