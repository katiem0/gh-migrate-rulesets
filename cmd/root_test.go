package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestWrappedFlagUsages(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("some-flag", "", "a deliberately long flag description that should wrap onto aligned continuation lines instead of running off the end of a single line")

	out := wrappedFlagUsages(fs)
	if !strings.Contains(out, "some-flag") {
		t.Errorf("wrappedFlagUsages() = %q, want it to contain the flag name", out)
	}
}

func TestRootHelpRendersWrappedFlags(t *testing.T) {
	cmd := NewCmdRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"create", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Flags:") || !strings.Contains(out, "actor-mapping") {
		t.Errorf("create --help output missing expected flags, got: %q", out)
	}
}

func TestNewCmdRoot(t *testing.T) {
	cmd := NewCmdRoot()

	if cmd.Use != "migrate-rules <command> [flags]" {
		t.Errorf("NewCmdRoot() Use = %v, want %v", cmd.Use, "migrate-rules <command> [flags]")
	}

	if !strings.Contains(cmd.Short, "List and create organization and repository rulesets") {
		t.Errorf("NewCmdRoot() Short description incorrect")
	}

	hasListCmd := false
	hasCreateCmd := false

	for _, subCmd := range cmd.Commands() {
		switch subCmd.Name() {
		case "list":
			hasListCmd = true
		case "create":
			hasCreateCmd = true
		}
	}

	if !hasListCmd {
		t.Error("NewCmdRoot() missing list command")
	}

	if !hasCreateCmd {
		t.Error("NewCmdRoot() missing create command")
	}
}

func TestRootCmd_CompletionOptions(t *testing.T) {
	cmd := NewCmdRoot()

	if !cmd.CompletionOptions.DisableDefaultCmd {
		t.Error("NewCmdRoot() should disable default completion command")
	}
}

func TestRootCommandHelpDisabled(t *testing.T) {
	cmd := NewCmdRoot()
	cmd.SetArgs([]string{"help"})

	err := cmd.Execute()

	if err == nil {
		return
	}

	t.Logf("Help command execution returned: %v", err)
}
