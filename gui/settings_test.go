package gui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettings_RoundTrip(t *testing.T) {
	isolateGUIEnvironment(t)

	if err := saveSettings(settings{Language: languageFrench}); err != nil {
		t.Fatalf("saveSettings: %v", err)
	}

	if got := loadSettings().Language; got != languageFrench {
		t.Errorf("language = %q, want %q", got, languageFrench)
	}
}

// saveSettings writes the whole struct and truncates the file, so writing one
// preference has to preserve the others. Without updateSettings, picking a
// language would erase what the last update check found, and the About tab
// would go back to GitHub on every launch.
func TestSettings_WritingOneFieldKeepsTheOthers(t *testing.T) {
	isolateGUIEnvironment(t)

	if err := updateSettings(func(s *settings) {
		s.UpdateCheckedAt = "2026-08-16T10:00:00Z"
		s.UpdateLatest = "0.4.0"
	}); err != nil {
		t.Fatalf("updateSettings: %v", err)
	}

	if err := updateSettings(func(s *settings) { s.Language = languageFrench }); err != nil {
		t.Fatalf("updateSettings: %v", err)
	}

	stored := loadSettings()
	if stored.Language != languageFrench {
		t.Errorf("language = %q, want %q", stored.Language, languageFrench)
	}
	if stored.UpdateLatest != "0.4.0" {
		t.Errorf("updateLatest = %q, want it preserved by the language write", stored.UpdateLatest)
	}
	if stored.UpdateCheckedAt == "" {
		t.Error("updateCheckedAt was lost by the language write")
	}

	// And the other way round.
	if err := updateSettings(func(s *settings) { s.UpdateLatest = "0.5.0" }); err != nil {
		t.Fatalf("updateSettings: %v", err)
	}
	if got := loadSettings().Language; got != languageFrench {
		t.Errorf("language = %q after an update write, want it preserved", got)
	}
}

// A first run has no preferences file, and that is not a failure: the frontend
// then follows the language Windows is running in.
func TestSettings_AbsentFileMeansNoPreference(t *testing.T) {
	isolateGUIEnvironment(t)

	if got := loadSettings().Language; got != "" {
		t.Errorf("language = %q on a first run, want \"\"", got)
	}
}

func TestSettings_ToleratesACorruptFile(t *testing.T) {
	isolateGUIEnvironment(t)

	path := settingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(path, []byte("{ not json"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if got := loadSettings().Language; got != "" {
		t.Errorf("language = %q from a corrupt file, want \"\"", got)
	}
}

// A language the interface is not translated into is worse than none: it would
// leave the user with a half-rendered catalogue.
func TestSettings_IgnoresAnUnsupportedLanguage(t *testing.T) {
	isolateGUIEnvironment(t)

	path := settingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"language":"de"}`), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if got := loadSettings().Language; got != "" {
		t.Errorf("language = %q, want it ignored", got)
	}
}

// The native dialogs are the one surface the frontend cannot draw, so they read
// the stored preference directly.
func TestDialogText_FollowsTheStoredLanguage(t *testing.T) {
	isolateGUIEnvironment(t)

	english := dialogText("dialog.open.title")

	if err := saveSettings(settings{Language: languageFrench}); err != nil {
		t.Fatalf("saveSettings: %v", err)
	}

	if french := dialogText("dialog.open.title"); french == english {
		t.Errorf("dialog caption stayed %q after switching to French", french)
	}
}
