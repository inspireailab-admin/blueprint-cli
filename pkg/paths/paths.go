// Package paths centralizes everything we put on the user's disk.
// Lives under ~/.blueprint (or the platform equivalent).
package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// Root returns the Blueprint home directory:
//
//   ~/.blueprint                  (Linux, macOS)
//   %USERPROFILE%\.blueprint      (Windows)
//
// Override with BLUEPRINT_HOME for development or unusual layouts.
func Root() (string, error) {
	if env := os.Getenv("BLUEPRINT_HOME"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate user home: %w", err)
	}
	return filepath.Join(home, ".blueprint"), nil
}

// Models is where GGUF weights live.
func Models() (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "models"), nil
}

// Bin is where the llama.cpp binaries live.
func Bin() (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "bin"), nil
}

// Runtime is where ephemeral state goes (pidfile, sockets, logs).
func Runtime() (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "runtime"), nil
}

// EnsureDir creates a directory (and parents) if it doesn't exist.
// Idempotent.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}
	return nil
}

// ModelFile returns the on-disk path for a given model + quant.
//
//   ~/.blueprint/models/<model-id>/<file-name>
func ModelFile(modelID, fileName string) (string, error) {
	models, err := Models()
	if err != nil {
		return "", err
	}
	return filepath.Join(models, modelID, fileName), nil
}
