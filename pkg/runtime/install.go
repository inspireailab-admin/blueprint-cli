package runtime

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/inspireailab-admin/blueprint/pkg/download"
	"github.com/inspireailab-admin/blueprint/pkg/paths"
)

// llamaCppRepo is the GitHub repo the runtime is fetched from.
const llamaCppRepo = "ggml-org/llama.cpp"

// userAgent is required by the GitHub API.
const userAgent = "blueprint-cli"

// versionFile records the llama.cpp tag installed in ~/.blueprint/bin/.
const versionFile = "VERSION"

// InstalledVersion returns the llama.cpp tag currently installed under
// ~/.blueprint/bin/, or "" if nothing's installed.
func InstalledVersion() string {
	bin, err := paths.Bin()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(bin, versionFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// InstallOptions controls Install behavior. Zero value is the legacy
// behavior — terminal progress on stderr, terminal log lines.
type InstallOptions struct {
	// OnProgress, when set, suppresses the stderr progress bar and
	// receives streaming download progress instead. Used by the
	// desktop app to forward bytes to a Wails event.
	OnProgress download.ProgressFunc

	// OnStage, when set, replaces the human-readable stderr log lines
	// ("Latest llama.cpp release: bXXX", "Extracting…") with structured
	// calls. Stages are: "locating", "downloading", "extracting", "done".
	OnStage func(stage string, detail string)
}

// Install fetches the latest llama.cpp release from GitHub and extracts
// the runtime binaries into ~/.blueprint/bin/. Convenience wrapper that
// uses the default stderr progress; see InstallWithOptions to route
// progress and stage notices elsewhere.
func Install(ctx context.Context) error {
	return InstallWithOptions(ctx, InstallOptions{})
}

// InstallWithOptions is the explicit-options variant. See InstallOptions.
func InstallWithOptions(ctx context.Context, opts InstallOptions) error {
	bin, err := paths.Bin()
	if err != nil {
		return err
	}
	if err := paths.EnsureDir(bin); err != nil {
		return err
	}

	emitStage := opts.OnStage
	if emitStage == nil {
		emitStage = func(_, _ string) {}
	}

	emitStage("locating", "")
	rel, err := fetchLatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("locate llama.cpp release: %w", err)
	}

	asset, err := pickAsset(rel.Assets)
	if err != nil {
		return err
	}

	if opts.OnStage == nil {
		fmt.Printf("Latest llama.cpp release: %s\n", rel.TagName)
		fmt.Printf("Picked asset: %s (%s)\n\n", asset.Name, humanBytes(asset.Size))
	}
	emitStage("downloading", fmt.Sprintf("%s (%s)", rel.TagName, asset.Name))

	archivePath := filepath.Join(bin, asset.Name)
	if err := download.FileWithOptions(ctx, asset.DownloadURL, archivePath, download.Options{OnProgress: opts.OnProgress}); err != nil {
		return fmt.Errorf("download asset: %w", err)
	}
	defer os.Remove(archivePath)

	if opts.OnStage == nil {
		fmt.Printf("\nExtracting runtime binaries…\n")
	}
	emitStage("extracting", asset.Name)

	count, err := extractRuntime(archivePath, bin)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	if err := os.WriteFile(filepath.Join(bin, versionFile), []byte(rel.TagName+"\n"), 0o644); err != nil {
		return fmt.Errorf("record version: %w", err)
	}

	if opts.OnStage == nil {
		fmt.Printf("✓ Installed llama.cpp %s (%d files) to %s\n", rel.TagName, count, bin)
		fmt.Printf("  Run with: blueprint serve <model-id>\n")
	}
	emitStage("done", fmt.Sprintf("%s · %d files", rel.TagName, count))
	return nil
}

// ─── GitHub releases ────────────────────────────────────────────────────────

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Name    string    `json:"name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

func fetchLatestRelease(ctx context.Context) (*ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", llamaCppRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github api returned %s: %s", resp.Status, string(body))
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// ─── Asset selection ────────────────────────────────────────────────────────

// pickAsset finds the right release archive for the running OS+arch.
// Thin wrapper around pickAssetFor for the production code path.
func pickAsset(assets []ghAsset) (ghAsset, error) {
	return pickAssetFor(assets, runtime.GOOS, runtime.GOARCH)
}

// pickAssetFor is the testable core. Given an asset list and an explicit
// (os, arch), pick the right release archive.
//
// Strategy is "prefer plain CPU / reference build":
//   - First pass: matching platform+arch AND no accelerator tag (cuda /
//     vulkan / hip / rocm / sycl / opencl / openvino). Apple Silicon CPU
//     builds already include Metal, so this gives GPU acceleration there
//     for free.
//   - Second pass: any platform+arch match, even an accelerator build.
//     Fallback for releases that only publish accelerator variants.
//
// llama.cpp publishes Windows builds as .zip and Mac / Linux builds as
// .tar.gz, so both extensions are accepted. `cudart-*` prefixed assets are
// CUDA runtime DLLs (not the runtime itself) and are skipped.
func pickAssetFor(assets []ghAsset, goos, goarch string) (ghAsset, error) {
	if len(assets) == 0 {
		return ghAsset{}, errors.New("release has no assets")
	}

	wantPlatform, wantArch := platformTagFor(goos, goarch)
	if wantPlatform == "" {
		return ghAsset{}, fmt.Errorf("unsupported OS %s/%s", goos, goarch)
	}

	matches := func(a ghAsset) bool {
		name := strings.ToLower(a.Name)
		if !isArchive(name) {
			return false
		}
		if strings.HasPrefix(name, "cudart-") {
			return false
		}
		if !strings.Contains(name, wantPlatform) {
			return false
		}
		if wantArch != "" && !strings.Contains(name, wantArch) {
			return false
		}
		return true
	}

	for _, a := range assets {
		if matches(a) && !hasAccelerator(strings.ToLower(a.Name)) {
			return a, nil
		}
	}
	for _, a := range assets {
		if matches(a) {
			return a, nil
		}
	}

	return ghAsset{}, fmt.Errorf("no release asset matches %s/%s — open an issue, this may be a new release naming pattern", goos, goarch)
}

func isArchive(name string) bool {
	return strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz")
}

// platformTag returns the substring pair (platform, arch) to look for in
// release asset names, for the running OS+arch.
func platformTag() (string, string) {
	return platformTagFor(runtime.GOOS, runtime.GOARCH)
}

// platformTagFor is the testable core. Returns ("", "") when we don't know
// how to handle the requested OS+arch.
func platformTagFor(goos, goarch string) (string, string) {
	switch goos {
	case "darwin":
		if goarch == "arm64" {
			return "macos-arm64", ""
		}
		return "macos-x64", ""
	case "linux":
		if goarch == "arm64" {
			return "ubuntu-arm64", ""
		}
		// ubuntu-22-x64 and similar — match on "ubuntu" + "x64"
		return "ubuntu", "x64"
	case "windows":
		if goarch == "arm64" {
			return "win", "arm64"
		}
		return "win", "x64"
	}
	return "", ""
}

// hasAccelerator returns true when the asset name encodes a hardware-specific
// build (CUDA / Vulkan / ROCm / SYCL / OpenCL / OpenVINO). We avoid these on
// auto-install for now — pinning to the reference CPU build keeps things
// portable, and on Apple Silicon Metal is built into the CPU asset anyway.
func hasAccelerator(name string) bool {
	for _, tag := range []string{
		"cuda", "vulkan", "hip", "rocm", "sycl",
		"kompute", "opencl", "openvino",
	} {
		if strings.Contains(name, tag) {
			return true
		}
	}
	return false
}

// ─── Extraction ─────────────────────────────────────────────────────────────

// extractRuntime dispatches to the right archive format and extracts everything
// under a `/bin/` directory (or top-level), flattened by basename, into target.
// Returns the count of files written. Binaries land 0755.
func extractRuntime(archivePath, target string) (int, error) {
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(archivePath, target)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(archivePath, target)
	}
	return 0, fmt.Errorf("unsupported archive: %s", archivePath)
}

func extractZip(archivePath, target string) (int, error) {
	rc, err := zip.OpenReader(archivePath)
	if err != nil {
		return 0, err
	}
	defer rc.Close()

	count := 0
	for _, f := range rc.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := normalizePath(f.Name)
		if !isRuntimeFile(name) {
			continue
		}
		dst := filepath.Join(target, filepath.Base(name))
		src, err := f.Open()
		if err != nil {
			return count, err
		}
		if err := writeRuntimeFile(dst, src); err != nil {
			src.Close()
			return count, err
		}
		src.Close()
		count++
	}
	if count == 0 {
		return 0, fmt.Errorf("zip contained no runtime files — release naming may have changed")
	}
	return count, nil
}

func extractTarGz(archivePath, target string) (int, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	count := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, err
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		name := normalizePath(hdr.Name)
		if !isRuntimeFile(name) {
			continue
		}
		dst := filepath.Join(target, filepath.Base(name))
		if err := writeRuntimeFile(dst, tr); err != nil {
			return count, err
		}
		count++
	}
	if count == 0 {
		return 0, fmt.Errorf("tarball contained no runtime files — release naming may have changed")
	}
	return count, nil
}

func normalizePath(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// isRuntimeFile decides whether an archive entry is a runtime artifact we
// want to extract. The Windows .zip is flat at the root (ggml.dll,
// llama-server.exe, …); the Mac / Linux tarballs nest things under build/bin/
// or bin/. Both shapes need to work.
func isRuntimeFile(name string) bool {
	// Block path traversal — defense in depth, archive/zip + archive/tar
	// already reject ".." in many cases, but be explicit.
	if strings.Contains(name, "..") {
		return false
	}
	// In a /bin/ segment — Mac / Linux tar.gz layout
	if strings.Contains(name, "/bin/") || strings.HasPrefix(name, "bin/") {
		return true
	}
	// Top-level (no directories) — Windows zip layout
	if !strings.Contains(name, "/") {
		return true
	}
	return false
}

func writeRuntimeFile(dst string, src io.Reader) error {
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

// humanBytes — small duplicate to avoid pulling internal/download into the
// public API surface of this package.
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
