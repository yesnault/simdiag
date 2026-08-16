package gui

import (
	"os"
	"path/filepath"
	"testing"

	"simdiag/common"
)

func TestSwitchTo_LoadsTheOtherConfiguration(t *testing.T) {
	isolateGUIEnvironment(t)

	state := newTestState(t, &common.Config{OutputDirectory: "./first"})

	dir := t.TempDir()
	other := filepath.Join(dir, configFileName)
	if err := common.SaveConfigTo(&common.Config{OutputDirectory: "./second"}, other); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := state.SwitchTo(other); err != nil {
		t.Fatalf("SwitchTo: %v", err)
	}

	if state.ConfigPath() != other {
		t.Errorf("ConfigPath() = %q, want %q", state.ConfigPath(), other)
	}
	if got := state.Config().OutputDirectory; got != "./second" {
		t.Errorf("OutputDirectory = %q, want the new file's value", got)
	}
}

// A configuration names its templates and output relative to itself, so the
// switch is only complete once the process has moved there too.
func TestSwitchTo_MovesToTheConfigurationDirectory(t *testing.T) {
	isolateGUIEnvironment(t)

	state := newTestState(t, &common.Config{})

	dir := t.TempDir()
	other := filepath.Join(dir, configFileName)
	if err := common.SaveConfigTo(&common.Config{}, other); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := state.SwitchTo(other); err != nil {
		t.Fatalf("SwitchTo: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if !sameDirectory(t, cwd, dir) {
		t.Errorf("working directory = %q, want the configuration's directory %q", cwd, dir)
	}
	if common.GetConfigFileName() != other {
		t.Errorf("common.ConfigFileName = %q, want %q", common.GetConfigFileName(), other)
	}
}

func TestSwitchTo_LeavesTheStateAloneOnAMissingFile(t *testing.T) {
	isolateGUIEnvironment(t)

	state := newTestState(t, &common.Config{OutputDirectory: "./kept"})
	before := state.ConfigPath()

	if err := state.SwitchTo(filepath.Join(t.TempDir(), "absent.yaml")); err == nil {
		t.Fatal("SwitchTo to a file that does not exist should fail")
	}

	if state.ConfigPath() != before {
		t.Errorf("ConfigPath() = %q after a failed switch, want the previous %q", state.ConfigPath(), before)
	}
	if got := state.Config().OutputDirectory; got != "./kept" {
		t.Errorf("OutputDirectory = %q after a failed switch, want the previous value", got)
	}
}

// sameDirectory compares two paths after resolving symlinks: on Windows a
// temporary directory is reached through a link often enough to matter.
func sameDirectory(t *testing.T, a, b string) bool {
	t.Helper()

	resolve := func(path string) string {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			return resolved
		}
		return path
	}

	return resolve(a) == resolve(b)
}
