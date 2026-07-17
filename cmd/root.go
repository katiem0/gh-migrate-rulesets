package cmd

import (
	"github.com/cli/go-gh/v2/pkg/term"
	createCmd "github.com/katiem0/gh-migrate-rulesets/cmd/create"
	listCmd "github.com/katiem0/gh-migrate-rulesets/cmd/list"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// defaultHelpWidth is used to wrap flag descriptions when the terminal width
// cannot be determined (e.g. output is piped or not a TTY).
const defaultHelpWidth = 80

// wrappedFlagUsages renders flag help wrapped to the current terminal width so
// long descriptions break onto aligned continuation lines instead of running
// off on a single line.
func wrappedFlagUsages(f *pflag.FlagSet) string {
	width := defaultHelpWidth
	if w, _, err := term.FromEnv().Size(); err == nil && w > 0 {
		width = w
	}
	return f.FlagUsagesWrapped(width)
}

// usageTemplate mirrors cobra's default template but wraps local and global flag
// descriptions to the terminal width via wrappedFlagUsages.
const usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{wrappedFlagUsages .LocalFlags | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{wrappedFlagUsages .InheritedFlags | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

func NewCmdRoot() *cobra.Command {
	cobra.AddTemplateFunc("wrappedFlagUsages", wrappedFlagUsages)

	cmdRoot := &cobra.Command{
		Use:   "migrate-rules <command> [flags]",
		Short: "List and create organization and repository rulesets.",
		Long:  "List and create repository/organization level rulesets for repositories in an organization.",
	}
	cmdRoot.SetUsageTemplate(usageTemplate)

	cmdRoot.AddCommand(listCmd.NewCmdList())
	cmdRoot.AddCommand(createCmd.NewCmdCreate())
	cmdRoot.CompletionOptions.DisableDefaultCmd = true
	cmdRoot.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
	return cmdRoot
}
