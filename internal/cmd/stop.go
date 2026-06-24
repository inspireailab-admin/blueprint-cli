package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop a running model server (coming next)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("stop is not implemented yet — coming in the next release")
		},
	}
}
