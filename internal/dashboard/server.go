// Package dashboard runs the local web UI for managing Blueprint.
//
// The dashboard is a tiny HTTP server that listens on 127.0.0.1 (loopback
// only — never reachable from the network) and serves an embedded SPA.
// It lives in the same binary as the rest of the CLI; users don't install
// it separately.
//
// PR 1 (this file): scaffolding only — serves the placeholder UI and a
// /api/health endpoint so we can verify wiring end-to-end.
// PR 2+: real model / metrics / chat / log APIs land on top of Handler.

package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/inspireailab-admin/blueprint/internal/dashboard/ui"
)

// Config controls the dashboard server. Zero values are sensible.
type Config struct {
	// Host is the bind host. Defaults to "127.0.0.1". Don't change this
	// unless you understand the implications — the dashboard has no auth
	// and assumes loopback-only reach.
	Host string

	// Port is the bind port. Defaults to 8081. Pass 0 to let the OS pick
	// a free one (useful in tests).
	Port int

	// Version is reported by /api/health. Wired from cmd.Version.
	Version string
}

func (c Config) host() string {
	if c.Host == "" {
		return "127.0.0.1"
	}
	return c.Host
}

func (c Config) port() int {
	if c.Port == 0 {
		return 8081
	}
	return c.Port
}

// Handler builds the dashboard's HTTP handler tree. Exposed so tests can
// drive it without binding a socket.
func Handler(cfg Config) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"version": cfg.Version,
		})
	})

	// Everything else falls through to the embedded SPA. http.FileServer
	// serves /index.html on /, /, /index.html, etc., which is what we want
	// for a single-page app.
	mux.Handle("/", http.FileServer(http.FS(ui.Assets())))

	return mux
}

// Run starts the dashboard server and blocks until ctx is canceled.
// Returns nil on clean shutdown, an error on bind / unexpected failure.
func Run(ctx context.Context, cfg Config) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.host(), cfg.port()))
	if err != nil {
		return fmt.Errorf("dashboard: listen: %w", err)
	}

	srv := &http.Server{
		Handler:           Handler(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Announce where we're listening — Run is called from the CLI, so this
	// is visible to the user and is what they paste into their browser if
	// the auto-open fails.
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return errors.New("dashboard: unexpected listener address type")
	}
	url := fmt.Sprintf("http://%s:%d", cfg.host(), addr.Port)
	fmt.Printf("→ Dashboard listening on %s\n", url)

	// Try to open the user's default browser. Best-effort: a headless box
	// or sandboxed environment will fail this silently; the URL above is
	// already on screen.
	go func() { _ = openBrowser(url) }()

	errCh := make(chan error, 1)
	go func() {
		err := srv.Serve(ln)
		if !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		<-errCh
		fmt.Println("Dashboard stopped.")
		return nil
	case err := <-errCh:
		return err
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
