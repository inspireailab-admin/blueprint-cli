// blueprint â€” the Inspire Blueprint CLI. Pulls open LLMs and runs them
// locally via llama.cpp. Drives a chat UI on localhost.
package main

import (
	"fmt"
	"os"

	"github.com/inspireailab-admin/blueprint-cli/internal/cmd"
)

func main() {
	if err := cmd.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
