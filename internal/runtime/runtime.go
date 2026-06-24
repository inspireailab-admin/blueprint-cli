// Package runtime locates and (later) installs the llama.cpp `llama-server` binary.
package runtime

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/inspireailab-admin/blueprint/internal/paths"
)

// ErrNotFound means llama-server isn't installed anywhere we can locate.
var ErrNotFound = errors.New("llama-server not found")

// binaryName is the executable filename on this OS.
func binaryName() string {
	if runtime.GOOS == "windows" {
		return "llama-server.exe"
	}
	return "llama-server"
}

// Find returns the path to llama-server.
//
// Lookup order:
//   1. ~/.blueprint/bin/llama-server[.exe]   (what `runtime install` lands)
//   2. $PATH                                 (Homebrew, system installs, etc.)
//
// Returns ErrNotFound when neither has it.
func Find() (string, error) {
	name := binaryName()

	// 1. Blueprint-managed install
	binDir, err := paths.Bin()
	if err == nil {
		candidate := filepath.Join(binDir, name)
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate, nil
		}
	}

	// 2. System PATH
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	return "", ErrNotFound
}

// InstallInstructions returns a short, per-OS string telling the user how to
// install llama-server. The Blueprint-managed install is always the first
// option; per-OS package managers come second for users who'd rather use
// the system.
func InstallInstructions() string {
	primary := "Blueprint-managed install (fastest):\n  blueprint runtime install"
	switch runtime.GOOS {
	case "darwin":
		return primary + "\n\nOr with Homebrew:\n  brew install llama.cpp"
	case "linux":
		return primary + "\n\nOr download a release zip from:\n  https://github.com/ggml-org/llama.cpp/releases/latest"
	case "windows":
		return primary + "\n\nOr with winget:\n  winget install llama.cpp"
	}
	return primary + fmt.Sprintf("\n\nOr for %s/%s, download from:\n  https://github.com/ggml-org/llama.cpp/releases/latest", runtime.GOOS, runtime.GOARCH)
}
