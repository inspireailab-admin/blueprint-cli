// Package download fetches files over HTTP with a progress bar and resume support.
package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// File downloads url to dst, showing progress on stderr. Resumes if a partial
// .part file exists. Atomic: renames .part → dst on success.
func File(ctx context.Context, url, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}

	part := dst + ".part"
	var existing int64
	if info, err := os.Stat(part); err == nil {
		existing = info.Size()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if existing > 0 {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(existing, 10)+"-")
	}

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status %d %s", resp.StatusCode, resp.Status)
	}

	var total int64
	if resp.ContentLength > 0 {
		total = resp.ContentLength
		if resp.StatusCode == http.StatusPartialContent {
			total += existing
		}
	}

	flag := os.O_CREATE | os.O_WRONLY
	if resp.StatusCode == http.StatusPartialContent {
		flag |= os.O_APPEND
	} else {
		// Server didn't support range — start fresh
		flag |= os.O_TRUNC
		existing = 0
	}
	out, err := os.OpenFile(part, flag, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	pw := &progressWriter{
		dst:     out,
		total:   total,
		written: existing,
		start:   time.Now(),
		label:   filepath.Base(dst),
	}
	if _, err := io.Copy(pw, resp.Body); err != nil {
		return err
	}
	pw.finish()

	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Rename(part, dst); err != nil {
		return fmt.Errorf("rename %s → %s: %w", part, dst, err)
	}
	return nil
}

// progressWriter renders a tiny ANSI progress line on stderr. No deps.
type progressWriter struct {
	dst        io.Writer
	total      int64
	written    int64
	start      time.Time
	label      string
	lastRender time.Time
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n, err := p.dst.Write(b)
	p.written += int64(n)
	if time.Since(p.lastRender) > 200*time.Millisecond {
		p.render()
		p.lastRender = time.Now()
	}
	return n, err
}

func (p *progressWriter) render() {
	elapsed := time.Since(p.start).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}
	speed := float64(p.written) / elapsed
	if p.total > 0 {
		pct := float64(p.written) / float64(p.total) * 100
		bar := makeBar(pct, 30)
		eta := time.Duration(float64(p.total-p.written)/speed) * time.Second
		fmt.Fprintf(os.Stderr, "\r%s  %s %5.1f%%  %s / %s  %s/s  ETA %s",
			p.label, bar, pct, humanBytes(p.written), humanBytes(p.total),
			humanBytes(int64(speed)), eta.Round(time.Second))
	} else {
		fmt.Fprintf(os.Stderr, "\r%s  %s  %s/s",
			p.label, humanBytes(p.written), humanBytes(int64(speed)))
	}
}

func (p *progressWriter) finish() {
	p.render()
	fmt.Fprintln(os.Stderr)
}

func makeBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct / 100 * float64(width))
	bar := make([]byte, width+2)
	bar[0] = '['
	for i := 0; i < width; i++ {
		if i < filled {
			bar[i+1] = '='
		} else if i == filled {
			bar[i+1] = '>'
		} else {
			bar[i+1] = ' '
		}
	}
	bar[width+1] = ']'
	return string(bar)
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

// ErrEmptyURL is returned when File is called with an empty URL.
var ErrEmptyURL = errors.New("empty URL")
