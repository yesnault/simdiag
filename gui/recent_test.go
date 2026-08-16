package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"simdiag/common"
)

// isolateGUIEnvironment keeps a test away from the developer's own recent-files
// list, and puts back the two process-wide things a configuration switch
// changes: the working directory and common.ConfigFileName.
//
// Anything using it cannot call t.Parallel, which is the point.
func isolateGUIEnvironment(t *testing.T) {
	t.Helper()

	settings := t.TempDir()
	t.Setenv("APPDATA", settings)         // os.UserConfigDir, Windows
	t.Setenv("XDG_CONFIG_HOME", settings) // os.UserConfigDir, elsewhere
	t.Chdir(t.TempDir())

	saved := common.GetConfigFileName()
	t.Cleanup(func() { common.SetConfigFileName(saved) })
}

// writeConfigFile creates an empty configuration and returns its path.
func writeConfigFile(t *testing.T, dir, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := common.SaveConfigTo(&common.Config{}, path); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return path
}

func TestRememberRecent_KeepsTheLastOpenedFirst(t *testing.T) {
	isolateGUIEnvironment(t)

	dir := t.TempDir()
	first := writeConfigFile(t, dir, "first.yaml")
	second := writeConfigFile(t, dir, "second.yaml")

	rememberRecent(first)
	rememberRecent(second)

	entries := loadRecent()
	if len(entries) != 2 {
		t.Fatalf("loadRecent() returned %d entries, want 2", len(entries))
	}
	if entries[0].Path != second {
		t.Errorf("loadRecent()[0] = %q, want the last one opened %q", entries[0].Path, second)
	}
	if entries[0].Name != "second.yaml" || entries[0].Dir != dir {
		t.Errorf("loadRecent()[0] = %+v, want it split into name and directory", entries[0])
	}
}

func TestRememberRecent_DeduplicatesIgnoringCase(t *testing.T) {
	isolateGUIEnvironment(t)

	path := writeConfigFile(t, t.TempDir(), "mapping_config.yaml")

	rememberRecent(path)
	rememberRecent(strings.ToUpper(path))

	if entries := loadRecent(); len(entries) != 1 {
		t.Fatalf("loadRecent() returned %d entries, want 1: the same file twice", len(entries))
	}
}

func TestRememberRecent_StopsAtTheLimit(t *testing.T) {
	isolateGUIEnvironment(t)

	dir := t.TempDir()
	for i := range recentLimit + 4 {
		rememberRecent(writeConfigFile(t, dir, "config-"+string(rune('a'+i))+".yaml"))
	}

	if entries := loadRecent(); len(entries) != recentLimit {
		t.Errorf("loadRecent() returned %d entries, want the limit of %d", len(entries), recentLimit)
	}
}

func TestLoadRecent_MarksFilesThatAreGone(t *testing.T) {
	isolateGUIEnvironment(t)

	path := writeConfigFile(t, t.TempDir(), "removed.yaml")
	rememberRecent(path)

	if err := os.Remove(path); err != nil {
		t.Fatalf("setup: %v", err)
	}

	entries := loadRecent()
	if len(entries) != 1 {
		t.Fatalf("loadRecent() returned %d entries, want 1", len(entries))
	}
	if !entries[0].Missing {
		t.Error("a deleted configuration should be reported as missing, not offered as openable")
	}
}

func TestMostRecentExisting_SkipsWhatIsGone(t *testing.T) {
	isolateGUIEnvironment(t)

	dir := t.TempDir()
	kept := writeConfigFile(t, dir, "kept.yaml")
	removed := writeConfigFile(t, dir, "removed.yaml")

	rememberRecent(kept)
	rememberRecent(removed)
	if err := os.Remove(removed); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if got := mostRecentExisting(); got != kept {
		t.Errorf("mostRecentExisting() = %q, want %q", got, kept)
	}
}

func TestMostRecentExisting_EmptyWithoutAList(t *testing.T) {
	isolateGUIEnvironment(t)

	if got := mostRecentExisting(); got != "" {
		t.Errorf("mostRecentExisting() = %q on a first run, want \"\"", got)
	}
}

// A corrupt list is a shortcut list, not the configuration: it must never stop
// the application from starting.
func TestLoadRecent_ToleratesACorruptList(t *testing.T) {
	isolateGUIEnvironment(t)

	listPath := recentListPath()
	if err := os.MkdirAll(filepath.Dir(listPath), 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(listPath, []byte("{ not json"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if entries := loadRecent(); len(entries) != 0 {
		t.Errorf("loadRecent() returned %d entries from a corrupt list, want none", len(entries))
	}
	if got := mostRecentExisting(); got != "" {
		t.Errorf("mostRecentExisting() = %q from a corrupt list, want \"\"", got)
	}
}
