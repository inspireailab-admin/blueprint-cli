package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/inspireailab-admin/blueprint/internal/dashboard"
)

func newDashboardCmd() *cobra.Command {
	var (
		host string
		port int
	)

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Open the Blueprint dashboard in your browser",
		Long: `Starts the Blueprint dashboard — a local web UI for managing
your private LLMs. Listens on 127.0.0.1 only (the internet
can't reach it) and opens your default browser when ready.

This is a v1 scaffold: real model, metrics, chat, and log
views land in the next few releases.

Ctrl-C to stop.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			return dashboard.Run(ctx, dashboard.Config{
				Host:    host,
				Port:    port,
				Version: Version,
			})
		},
	}

	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "bind host (do not change unless you know what you're doing — no auth)")
	cmd.Flags().IntVar(&port, "port", 8081, "HTTP port to listen on")
	return cmd
}
