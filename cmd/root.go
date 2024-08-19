package cmd

import (
	createCmd "github.com/katiem0/gh-migrate-rulesets/cmd/create"
	listCmd "github.com/katiem0/gh-migrate-rulesets/cmd/list"
	"github.com/spf13/cobra"
)

func NewCmdRoot() *cobra.Command {

	cmdRoot := &cobra.Command{
		Use:   "migrate-rules <command> [flags]",
		Short: "List and create organization and repository rulesets.",
		Long:  "List and create repository/organization level rulesets for repositories in an organization.",
	}

	cmdRoot.AddCommand(listCmd.NewCmdList())
	cmdRoot.AddCommand(createCmd.NewCmdCreate())
	cmdRoot.CompletionOptions.DisableDefaultCmd = true
	cmdRoot.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
	return cmdRoot
}
