// Package cmd defines the gh repopotamus command tree.
package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "repopotamus",
	Short:         "Move GitHub repos from one owner to another",
	SilenceUsage:  true,
	SilenceErrors: false,
}

func Execute() error {
	return rootCmd.Execute()
}
