// Package catalog is the CLI-side view of the Inspire Blueprint model catalog.
// The full curated catalog ships from the web app's data/models.json; this
// package exposes the slice the CLI needs (id → GGUF URL) using a snapshot
// that's regenerated at release time via `go generate`.
package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed models.json
var raw []byte

// Model is the trimmed view of the catalog the CLI needs.
type Model struct {
	ID          string            `json:"id"`
	DisplayName string            `json:"displayName"`
	Family      string            `json:"family"`
	Params      float64           `json:"params"`
	TotalParams float64           `json:"totalParams"`
	License     string            `json:"license"`
	GgufRepo    string            `json:"ggufRepo,omitempty"`
	GgufFiles   map[string]string `json:"ggufFiles,omitempty"`
}

// Load parses the embedded catalog. Cheap, call freely.
func Load() ([]Model, error) {
	var models []Model
	if err := json.Unmarshal(raw, &models); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	return models, nil
}

// Get returns the model with the given id, or an error if absent.
func Get(id string) (Model, error) {
	models, err := Load()
	if err != nil {
		return Model{}, err
	}
	for _, m := range models {
		if m.ID == id {
			return m, nil
		}
	}
	return Model{}, fmt.Errorf("unknown model %q (run `blueprint pull` with no args to list available)", id)
}

// DownloadURL builds the HuggingFace resolve URL for the given quant, or
// returns an error if the model has no GGUF source or the quant isn't
// published.
func (m Model) DownloadURL(quant string) (string, string, error) {
	if m.GgufRepo == "" || m.GgufFiles == nil {
		return "", "", fmt.Errorf("model %s isn't available for local install yet", m.ID)
	}
	file, ok := m.GgufFiles[quant]
	if !ok {
		return "", "", fmt.Errorf("model %s has no %s GGUF (available: %v)", m.ID, quant, availableQuants(m.GgufFiles))
	}
	return fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", m.GgufRepo, file), file, nil
}

func availableQuants(files map[string]string) []string {
	out := make([]string, 0, len(files))
	for q := range files {
		out = append(out, q)
	}
	return out
}
