package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/inspireailab-admin/blueprint/pkg/catalog"
	"github.com/inspireailab-admin/blueprint/pkg/download"
	"github.com/inspireailab-admin/blueprint/pkg/paths"
)

func newPullCmd() *cobra.Command {
	var quant string

	cmd := &cobra.Command{
		Use:   "pull [model-id]",
		Short: "Download a model's GGUF weights to ~/.blueprint/models",
		Long: `Downloads the GGUF weights for a model so it can be served locally.

With no arguments, lists the models available to pull. With a model id,
fetches the GGUF for the given quant (default q4) into:

  ~/.blueprint/models/<model-id>/<file>.gguf

The download is resumable — if it's interrupted, re-running the same
command picks up where it left off.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return listModels()
			}
			return pullModel(args[0], quant)
		},
	}

	cmd.Flags().StringVarP(&quant, "quant", "q", "q4", "weight quantization to fetch (q3, q4, q8, fp16)")
	return cmd
}

func listModels() error {
	models, err := catalog.Load()
	if err != nil {
		return err
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })

	fmt.Println("Available models (use `blueprint pull <id>`):")
	fmt.Println()
	for _, m := range models {
		quants := make([]string, 0, len(m.GgufFiles))
		for q := range m.GgufFiles {
			quants = append(quants, q)
		}
		sort.Strings(quants)
		fmt.Printf("  %-32s %s · %sB · %s · quants: %s\n",
			m.ID, m.Family, fmtParams(m.Params), m.License, strings.Join(quants, ", "))
	}
	fmt.Println()
	fmt.Println("Default quant is q4. Override with --quant q8 (etc).")
	return nil
}

func pullModel(id, quant string) error {
	model, err := catalog.Get(id)
	if err != nil {
		return err
	}
	url, fileName, err := model.DownloadURL(quant)
	if err != nil {
		return err
	}

	dst, err := paths.ModelFile(model.ID, fileName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		fmt.Printf("✓ already on disk: %s\n", dst)
		return nil
	}

	fmt.Printf("Pulling %s (%s)\n  from %s\n  to   %s\n\n", model.DisplayName, quant, url, dst)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := download.File(ctx, url, dst); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("download interrupted (re-run to resume)")
		}
		return fmt.Errorf("download: %w", err)
	}

	fmt.Printf("\n✓ %s\n  Run with: blueprint serve %s\n", dst, model.ID)
	return nil
}

func fmtParams(p float64) string {
	if p == float64(int(p)) {
		return fmt.Sprintf("%d", int(p))
	}
	return fmt.Sprintf("%.1f", p)
}
