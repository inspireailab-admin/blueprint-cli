package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/inspireailab-admin/blueprint-cli/pkg/paths"
	"github.com/inspireailab-admin/blueprint-cli/pkg/runtime"
)

func newRuntimeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "runtime",
		Short: "Manage the llama.cpp runtime (llama-server)",
		Long: `Blueprint runs models through llama.cpp's llama-server. Use these
subcommands to install or inspect the runtime.

The runtime is installed under ~/.blueprint/bin/. It's separate from
any system-wide installation you may have (Homebrew, winget). When
both exist, the Blueprint-managed one wins.`,
	}
	c.AddCommand(newRuntimeInstallCmd())
	c.AddCommand(newRuntimeVersionCmd())
	c.AddCommand(newRuntimePathCmd())
	return c
}

func newRuntimeInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Download and install llama-server (latest llama.cpp release)",
		Long: `Pulls the latest llama.cpp release from GitHub and extracts the
runtime binaries (llama-server + supporting libraries) into
~/.blueprint/bin/. Re-running upgrades to the latest version.

This installs the reference CPU build â€” on Apple Silicon it
already includes Metal acceleration. Dedicated CUDA / Vulkan
variants for NVIDIA / AMD GPUs are a flag we'll add next.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return runtime.Install(ctx)
		},
	}
}

func newRuntimeVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the installed llama.cpp version",
		RunE: func(_ *cobra.Command, _ []string) error {
			v := runtime.InstalledVersion()
			if v == "" {
				fmt.Println("llama.cpp is not installed via Blueprint.")
				fmt.Println("Install it with: blueprint runtime install")
				return nil
			}
			fmt.Println(v)
			return nil
		},
	}
}

func newRuntimePathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the resolved path to llama-server",
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := runtime.Find()
			if err != nil {
				bin, _ := paths.Bin()
				fmt.Fprintf(os.Stderr, "llama-server not found.\n")
				fmt.Fprintf(os.Stderr, "Blueprint would install it to: %s\n", bin)
				fmt.Fprintf(os.Stderr, "Run: blueprint runtime install\n")
				return err
			}
			fmt.Println(p)
			return nil
		},
	}
}
