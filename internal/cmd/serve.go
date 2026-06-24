package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/inspireailab-admin/blueprint/pkg/catalog"
	"github.com/inspireailab-admin/blueprint/pkg/paths"
	"github.com/inspireailab-admin/blueprint/pkg/runtime"
)

// Auth key the web UI uses to reach the local server. The browser sends this
// in the Authorization header; the user never types it. A fixed value is fine
// because the server only listens on 127.0.0.1.
const localAPIKey = "blueprint-local"

func newServeCmd() *cobra.Command {
	var (
		quant   string
		port    int
		nGPU    int
		nThread int
		ctxSize int
	)

	cmd := &cobra.Command{
		Use:   "serve [model-id]",
		Short: "Run a pulled model with llama-server",
		Long: `Spawns llama.cpp's llama-server against a previously-pulled model
and exposes an OpenAI-compatible API at:

  http://127.0.0.1:<port>/v1
  Authorization: Bearer ` + localAPIKey + `

Stops cleanly on Ctrl-C. The Blueprint web UI knows how to find this
endpoint and drive a chat session against it.

llama.cpp is not bundled — install it once:

  blueprint runtime install        (Blueprint-managed, ~/.blueprint/bin/)
  brew install llama.cpp           (macOS, system-wide)
  winget install llama.cpp         (Windows, system-wide)`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return serveModel(args[0], serveOpts{
				quant:   quant,
				port:    port,
				nGPU:    nGPU,
				nThread: nThread,
				ctxSize: ctxSize,
			})
		},
	}

	cmd.Flags().StringVarP(&quant, "quant", "q", "q4", "weight quant to serve (must match what was pulled)")
	cmd.Flags().IntVar(&port, "port", 8080, "HTTP port to listen on (127.0.0.1)")
	cmd.Flags().IntVar(&nGPU, "n-gpu-layers", 999, "layers to offload to GPU (0 = CPU only)")
	cmd.Flags().IntVar(&nThread, "threads", 0, "CPU threads (0 = llama-server default)")
	cmd.Flags().IntVar(&ctxSize, "ctx-size", 4096, "context window in tokens")
	return cmd
}

type serveOpts struct {
	quant   string
	port    int
	nGPU    int
	nThread int
	ctxSize int
}

func serveModel(id string, opts serveOpts) error {
	model, err := catalog.Get(id)
	if err != nil {
		return err
	}

	// Find the pulled GGUF on disk
	fileName, ok := model.GgufFiles[opts.quant]
	if !ok {
		return fmt.Errorf("model %s has no %s GGUF in our catalog", model.ID, opts.quant)
	}
	modelPath, err := paths.ModelFile(model.ID, fileName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(modelPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("model not on disk: %s\n  pull it first: blueprint pull %s --quant %s", modelPath, model.ID, opts.quant)
	}

	// Find llama-server
	bin, err := runtime.Find()
	if err != nil {
		return fmt.Errorf("%w\n\n%s", err, runtime.InstallInstructions())
	}

	args := buildArgs(modelPath, opts)

	fmt.Printf("Starting llama-server\n")
	fmt.Printf("  binary : %s\n", bin)
	fmt.Printf("  model  : %s\n", filepath.Base(modelPath))
	fmt.Printf("  endpoint: http://127.0.0.1:%d/v1\n", opts.port)
	fmt.Printf("  api key : %s\n", localAPIKey)
	fmt.Printf("  ctx size: %d  ·  gpu layers: %d  ·  threads: %d\n\n", opts.ctxSize, opts.nGPU, opts.nThread)
	fmt.Printf("Ctrl-C to stop. Waiting for ready…\n\n")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	c := exec.CommandContext(ctx, bin, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Start(); err != nil {
		return fmt.Errorf("start llama-server: %w", err)
	}

	// Watch readiness in parallel so we can announce when the API is up
	readyCh := make(chan error, 1)
	go func() { readyCh <- waitForReady(ctx, opts.port, 60*time.Second) }()

	// Wait for the subprocess to exit (or for the user to Ctrl-C)
	waitErr := c.Wait()

	// Drain readiness result for tidy logs
	select {
	case <-readyCh:
	default:
	}

	if ctx.Err() != nil {
		fmt.Println("Stopped.")
		return nil
	}
	if waitErr != nil {
		return fmt.Errorf("llama-server exited: %w", waitErr)
	}
	return nil
}

func buildArgs(modelPath string, opts serveOpts) []string {
	args := []string{
		"--model", modelPath,
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(opts.port),
		"--api-key", localAPIKey,
		"--ctx-size", strconv.Itoa(opts.ctxSize),
		"--n-gpu-layers", strconv.Itoa(opts.nGPU),
	}
	if opts.nThread > 0 {
		args = append(args, "--threads", strconv.Itoa(opts.nThread))
	}
	return args
}

// waitForReady polls /health until it returns 200, the timeout fires, or the
// context is canceled. We don't fail serve when it times out — llama-server
// can take a while to load big models — we just stop announcing.
func waitForReady(ctx context.Context, port int, timeout time.Duration) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	deadline := time.Now().Add(timeout)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(deadline) {
			return errors.New("ready probe timed out")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+localAPIKey)
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					fmt.Printf("✓ ready at http://127.0.0.1:%d/v1\n\n", port)
					return nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}
