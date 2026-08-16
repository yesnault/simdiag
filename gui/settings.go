package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Languages the interface is available in.
const (
	languageEnglish = "en"
	languageFrench  = "fr"
)

// settings holds what belongs to the user rather than to a configuration file.
//
// It lives next to recent.json for the same reason that list does: a preference
// spans configuration files, so it cannot live inside one of them. Opening
// another pilot's profile must not change the language of the interface.
type settings struct {
	Language string `json:"language,omitempty"`

	// What the last look at GitHub found. Remembering it is what lets the tab
	// badge appear at once on startup instead of waiting on a round trip, and it
	// spares the 60 requests an hour GitHub grants without authentication.
	UpdateCheckedAt string `json:"updateCheckedAt,omitempty"` // RFC3339
	UpdateLatest    string `json:"updateLatest,omitempty"`
}

// settingsPath returns %APPDATA%\simdiag\settings.json.
func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "simdiag", "settings.json")
}

// loadSettings reads the user's preferences.
//
// Nothing here is fatal: a missing, unreadable or corrupt file simply means the
// user has expressed no preference yet, which is also the state of a first run.
func loadSettings() settings {
	path := settingsPath()
	if path == "" {
		return settings{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return settings{}
	}

	var loaded settings
	if err := json.Unmarshal(data, &loaded); err != nil {
		return settings{}
	}

	if !isSupportedLanguage(loaded.Language) {
		loaded.Language = ""
	}

	return loaded
}

// saveSettings writes the user's preferences.
//
// Unlike rememberRecent, a failure is reported: this runs in answer to the user
// clicking FR or EN, and silently forgetting the choice would look like a bug
// the next time the application starts.
func saveSettings(updated settings) error {
	path := settingsPath()
	if path == "" {
		return fmt.Errorf("no location to store the preferences")
	}

	data, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("unable to create %s: %w", filepath.Dir(path), err)
	}

	return os.WriteFile(path, data, 0644)
}

// updateSettings applies a change on top of what is already stored.
//
// saveSettings writes the whole struct and truncates the file, so writing one
// field means reading the others back first. Otherwise picking a language would
// erase what the last update check found, and vice versa.
func updateSettings(apply func(*settings)) error {
	current := loadSettings()
	apply(&current)
	return saveSettings(current)
}

// isSupportedLanguage reports whether the interface is translated into lang.
func isSupportedLanguage(lang string) bool {
	return lang == languageEnglish || lang == languageFrench
}
