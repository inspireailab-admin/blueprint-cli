package catalog

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	models, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("catalog is empty")
	}

	// Sanity: every entry has the minimum fields the CLI needs to do anything
	seen := map[string]bool{}
	for _, m := range models {
		if m.ID == "" {
			t.Errorf("model has empty id: %+v", m)
		}
		if seen[m.ID] {
			t.Errorf("duplicate id %q", m.ID)
		}
		seen[m.ID] = true
		if m.DisplayName == "" {
			t.Errorf("%s: missing displayName", m.ID)
		}
		if m.Family == "" {
			t.Errorf("%s: missing family", m.ID)
		}
		if m.GgufRepo == "" {
			t.Errorf("%s: missing ggufRepo", m.ID)
		}
		if len(m.GgufFiles) == 0 {
			t.Errorf("%s: no GGUF files declared", m.ID)
		}
		for quant, file := range m.GgufFiles {
			if file == "" {
				t.Errorf("%s: empty filename for quant %q", m.ID, quant)
			}
			if !strings.HasSuffix(file, ".gguf") {
				t.Errorf("%s: quant %q file %q doesn't end in .gguf", m.ID, quant, file)
			}
		}
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"known id", "qwen-2.5-7b-instruct", false},
		{"unknown id", "definitely-not-a-real-model", true},
		{"empty id", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := Get(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Get(%q) = %+v, want error", tt.id, m)
				}
				return
			}
			if err != nil {
				t.Errorf("Get(%q) error = %v", tt.id, err)
			}
			if m.ID != tt.id {
				t.Errorf("Get(%q).ID = %q", tt.id, m.ID)
			}
		})
	}
}

func TestDownloadURL(t *testing.T) {
	m, err := Get("qwen-2.5-7b-instruct")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	t.Run("known quant", func(t *testing.T) {
		url, file, err := m.DownloadURL("q4")
		if err != nil {
			t.Fatalf("DownloadURL(q4): %v", err)
		}
		if !strings.HasPrefix(url, "https://huggingface.co/") {
			t.Errorf("url = %q, want https://huggingface.co/ prefix", url)
		}
		if !strings.Contains(url, m.GgufRepo) {
			t.Errorf("url = %q, want it to contain %q", url, m.GgufRepo)
		}
		if !strings.HasSuffix(url, file) {
			t.Errorf("url = %q, file = %q — url should end with file", url, file)
		}
	})

	t.Run("unknown quant returns clear error", func(t *testing.T) {
		_, _, err := m.DownloadURL("q99")
		if err == nil {
			t.Fatal("DownloadURL(q99): want error, got nil")
		}
		if !strings.Contains(err.Error(), "q99") {
			t.Errorf("error should mention requested quant: %v", err)
		}
	})

	t.Run("model with no GGUF returns clear error", func(t *testing.T) {
		empty := Model{ID: "test"}
		_, _, err := empty.DownloadURL("q4")
		if err == nil {
			t.Fatal("DownloadURL on empty model: want error, got nil")
		}
		if !strings.Contains(err.Error(), "local install") {
			t.Errorf("error should explain local install isn't available: %v", err)
		}
	})
}
