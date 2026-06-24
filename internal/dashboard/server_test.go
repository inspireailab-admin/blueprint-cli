package dashboard

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Handler should serve /api/health as JSON with the configured version.
func TestHandlerHealth(t *testing.T) {
	h := Handler(Config{Version: "test-1.2.3"})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type: got %q, want application/json", ct)
	}

	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status field: got %q, want %q", body.Status, "ok")
	}
	if body.Version != "test-1.2.3" {
		t.Errorf("version field: got %q, want %q", body.Version, "test-1.2.3")
	}
}

// Handler should serve the embedded index.html at the root path. We don't
// assert the full body — the placeholder content can drift — but the
// response should be HTML and mention "Blueprint" somewhere.
func TestHandlerServesEmbeddedIndex(t *testing.T) {
	h := Handler(Config{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("content-type: got %q, want text/html", ct)
	}
	body, _ := io.ReadAll(rr.Body)
	if !strings.Contains(string(body), "Blueprint") {
		t.Errorf("embedded index.html doesn't mention Blueprint — embedding broken?")
	}
}

// Run binds, serves, and shuts down cleanly when ctx is canceled. Uses
// port 0 to avoid clashing with anything the dev machine has running.
func TestRunStartStop(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("probe listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Config{Port: port, Version: "test"})
	}()

	// Wait for the server to come up. /api/health is our liveness probe.
	url := "http://127.0.0.1:" + itoa(port) + "/api/health"
	if !waitForOK(url, 3*time.Second) {
		t.Fatalf("server never became reachable at %s", url)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error on shutdown: %v", err)
		}
	case <-time.After(7 * time.Second):
		t.Fatalf("Run didn't return after ctx cancel")
	}
}

func waitForOK(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// strconv.Itoa kept out so test imports stay tight.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
