package paths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoot_HonorsBlueprintHome(t *testing.T) {
	t.Setenv("BLUEPRINT_HOME", "/custom/path")
	got, err := Root()
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	if got != "/custom/path" {
		t.Errorf("Root = %q, want /custom/path", got)
	}
}

func TestRoot_FallsBackToHomeDotBlueprint(t *testing.T) {
	t.Setenv("BLUEPRINT_HOME", "")
	got, err := Root()
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".blueprint")
	if got != want {
		t.Errorf("Root = %q, want %q", got, want)
	}
}

func TestSubdirsAreUnderRoot(t *testing.T) {
	t.Setenv("BLUEPRINT_HOME", "/anchor")

	cases := []struct {
		name string
		fn   func() (string, error)
		want string
	}{
		{"Models", Models, filepath.Join("/anchor", "models")},
		{"Bin", Bin, filepath.Join("/anchor", "bin")},
		{"Runtime", Runtime, filepath.Join("/anchor", "runtime")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.fn()
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			if got != c.want {
				t.Errorf("%s = %q, want %q", c.name, got, c.want)
			}
		})
	}
}

func TestEnsureDir_IsIdempotent(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "a", "b", "c")
	for i := 0; i < 3; i++ {
		if err := EnsureDir(target); err != nil {
			t.Fatalf("call %d: EnsureDir(%q): %v", i, target, err)
		}
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("%q exists but is not a directory", target)
	}
}

func TestModelFile_BuildsExpectedPath(t *testing.T) {
	t.Setenv("BLUEPRINT_HOME", "/anchor")
	got, err := ModelFile("qwen-2.5-7b-instruct", "Q4_K_M.gguf")
	if err != nil {
		t.Fatalf("ModelFile: %v", err)
	}
	// Path-separator-agnostic substring checks (Windows vs Unix)
	for _, want := range []string{"qwen-2.5-7b-instruct", "Q4_K_M.gguf", "models"} {
		if !strings.Contains(got, want) {
			t.Errorf("ModelFile = %q, expected to contain %q", got, want)
		}
	}
}
