package cmd

import (
	"github.com/spf13/cobra"
)

// Version is the CLI version. Override with `-ldflags "-X .../cmd.Version=..."` at build time.
var Version = "0.0.1-dev"

// NewRoot wires the top-level command and all subcommands.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "blueprint",
		Short: "Inspire Blueprint — pull and run open LLMs locally",
		Long: `Blueprint is the local-install companion to the Inspire Blueprint
planning tool. It downloads open-model GGUF weights and runs them
through llama.cpp on your machine, exposing an OpenAI-compatible
endpoint that the Blueprint web UI can drive a chat session against.

Free, no telemetry, no account. Made by Inspire AI Lab.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newVersionCmd())
	root.AddCommand(newPullCmd())
	root.AddCommand(newServeCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newStopCmd())
	root.AddCommand(newRuntimeCmd())

	return root
}
