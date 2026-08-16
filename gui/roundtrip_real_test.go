package gui

import (
	"os"
	"path/filepath"
	"testing"

	"simdiag/common"
)

// TestRealConfigSurvivesAGUISave loads the repository's own mapping_config.yaml,
// runs it through the Configuration form projection and writes it back, then
// compares the YAML byte for byte. Saving without editing anything must be a
// no-op, otherwise the GUI silently rewrites users' configurations.
func TestRealConfigSurvivesAGUISave(t *testing.T) {
	source := filepath.Join("..", "mapping_config.yaml")
	if _, err := os.Stat(source); err != nil {
		t.Skip("no mapping_config.yaml at the repository root")
	}

	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	config, err := common.LoadConfigFrom(source)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	applyDTO(config, toDTO(config))

	out := filepath.Join(t.TempDir(), "mapping_config.yaml")
	if err := common.SaveConfigTo(config, out); err != nil {
		t.Fatalf("save: %v", err)
	}

	saved, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	if string(saved) != string(original) {
		t.Errorf("a no-op GUI save rewrote the configuration.\n--- original (%d bytes)\n--- saved (%d bytes)", len(original), len(saved))
	}
}
