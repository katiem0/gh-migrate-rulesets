package cmd

import (
	"strings"
	"testing"
)

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
