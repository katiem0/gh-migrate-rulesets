package main

import (
	"os"
	"testing"
)

// TestMainHelp exercises main() via the --help path, which runs the root command
// to completion and returns without calling os.Exit.
func TestMainHelp(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"migrate-rules", "--help"}
	main()
}
