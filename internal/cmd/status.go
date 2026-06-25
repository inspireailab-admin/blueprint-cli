package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/inspireailab-admin/blueprint-cli/pkg/paths"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show installed models and running servers",
		RunE: func(_ *cobra.Command, _ []string) error {
			root, err := paths.Root()
			if err != nil {
				return err
			}
			fmt.Println("Blueprint home:", root)
			fmt.Println()

			models, err := paths.Models()
			if err != nil {
				return err
			}
			entries, err := os.ReadDir(models)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No models pulled yet. Run `blueprint pull` to see what's available.")
					return nil
				}
				return err
			}

			fmt.Println("Pulled models:")
			ids := make([]string, 0, len(entries))
			for _, e := range entries {
				if e.IsDir() {
					ids = append(ids, e.Name())
				}
			}
			sort.Strings(ids)

			if len(ids) == 0 {
				fmt.Println("  (none)")
				return nil
			}
			for _, id := range ids {
				dir := filepath.Join(models, id)
				files, _ := os.ReadDir(dir)
				for _, f := range files {
					if f.IsDir() {
						continue
					}
					info, err := f.Info()
					if err != nil {
						continue
					}
					fmt.Printf("  %s  %s  (%s)\n", id, f.Name(), humanBytes(info.Size()))
				}
			}
			return nil
		},
	}
}

func humanBytes(n int64) string {
	const k = 1024
	if n < k {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(k), 0
	for x := n / k; x >= k; x /= k {
		div *= k
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
