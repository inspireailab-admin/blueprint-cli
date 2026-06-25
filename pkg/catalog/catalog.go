// Package catalog is the canonical Inspire Blueprint model catalog â€”
// model IDs, architecture fields needed for VRAM sizing, license,
// capability flags, and the GGUF download metadata that powers
// `blueprint pull`.
//
// The catalog ships as an embedded JSON snapshot. The marketing site
// at inspireailab.com syncs its own copy down from this package's
// models.json via the URL
// https://raw.githubusercontent.com/inspireailab-admin/blueprint-cli/main/pkg/catalog/models.json
// â€” kernel is source of truth.

package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed models.json
var raw []byte

// Catalog is the full embedded payload: the model list plus the
// metadata the marketing site surfaces as the "estimates as of" date.
type Catalog struct {
	AsOf   string  `json:"asOf"`
	Note   string  `json:"note"`
	Models []Model `json:"models"`
}

// Model is one entry in the catalog. Field set mirrors the marketing
// site's TypeScript Model type so the desktop app's React layer can
// consume it 1:1 over Wails IPC without a translation step.
type Model struct {
	ID                string        `json:"id"`
	DisplayName       string        `json:"displayName"`
	Family            string        `json:"family"`
	Params            float64       `json:"params"`
	TotalParams       float64       `json:"totalParams"`
	Type              string        `json:"type"`
	License           string        `json:"license"`
	Gated             bool          `json:"gated"`
	MaxContext        int           `json:"maxContext"`
	IsMoE             bool          `json:"isMoE"`
	QuantOptions      []string      `json:"quantOptions"`
	NumLayers         int           `json:"numLayers"`
	NumKvHeads        int           `json:"numKvHeads"`
	HiddenSize        int           `json:"hiddenSize"`
	NumAttentionHeads int           `json:"numAttentionHeads"`
	Capabilities      Capabilities  `json:"capabilities,omitempty"`
	PopularityRank    int           `json:"popularityRank,omitempty"`
	Local             *LocalInstall `json:"local,omitempty"`
}

// Capabilities flags surface in the planner UI as filterable / scorable
// attributes. All optional â€” absent means "not declared," not "no."
type Capabilities struct {
	StructuredOutput  bool `json:"structuredOutput,omitempty"`
	Multilingual      bool `json:"multilingual,omitempty"`
	LongContextProven bool `json:"longContextProven,omitempty"`
}

// LocalInstall carries the metadata needed to install this model on
// the user's machine via the CLI or the desktop app. Models with
// Available=false are catalog-visible (you can size them) but not
// installable through Blueprint yet.
type LocalInstall struct {
	Available bool              `json:"available"`
	GgufRepo  string            `json:"ggufRepo,omitempty"`
	GgufFiles map[string]string `json:"ggufFiles,omitempty"`
}

// LoadFull parses the embedded catalog including the metadata header.
// Use this when you need the AsOf / Note fields; LoadModels is enough
// for the common "give me the slice" case.
func LoadFull() (Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		return Catalog{}, fmt.Errorf("parse catalog: %w", err)
	}
	return c, nil
}

// Load returns just the slice of models. Kept as the primary entry
// point because most callers (CLI commands, tests) don't need the
// metadata. Cheap â€” call freely; parsing is pure JSON over an
// embedded byte slice.
func Load() ([]Model, error) {
	c, err := LoadFull()
	if err != nil {
		return nil, err
	}
	return c.Models, nil
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

// IsInstallable reports whether the model can be pulled and served
// today. Uses the same condition as DownloadURL â€” Local present + a
// GgufRepo set.
func (m Model) IsInstallable() bool {
	return m.Local != nil && m.Local.Available && m.Local.GgufRepo != ""
}

// QuantFiles returns the map of available GGUF quants â†’ file names,
// or nil if the model isn't installable.
func (m Model) QuantFiles() map[string]string {
	if m.Local == nil {
		return nil
	}
	return m.Local.GgufFiles
}

// DownloadURL builds the HuggingFace resolve URL for the given quant,
// or returns an error if the model has no GGUF source or the quant
// isn't published.
func (m Model) DownloadURL(quant string) (string, string, error) {
	if !m.IsInstallable() {
		return "", "", fmt.Errorf("model %s isn't available for local install yet", m.ID)
	}
	file, ok := m.Local.GgufFiles[quant]
	if !ok {
		return "", "", fmt.Errorf("model %s has no %s GGUF (available: %v)", m.ID, quant, availableQuants(m.Local.GgufFiles))
	}
	return fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", m.Local.GgufRepo, file), file, nil
}

func availableQuants(files map[string]string) []string {
	out := make([]string, 0, len(files))
	for q := range files {
		out = append(out, q)
	}
	return out
}
