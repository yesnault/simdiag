package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// recentLimit is how many configuration files the picker remembers. The list is
// a shortcut, not a history: past a handful of entries the menu is harder to
// read than the file dialog it replaces.
const recentLimit = 8

// recentEntry is one remembered configuration file, pre-split for display: the
// menu shows the file name on one line and its directory underneath, since
// several profiles are usually all called mapping_config.yaml.
type recentEntry struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Dir     string `json:"dir"`
	Missing bool   `json:"missing"` // the file is gone; shown, but not selectable
}

// recentListPath returns %APPDATA%\simdiag\recent.json.
//
// The list is deliberately kept outside any configuration directory: it spans
// them, and the whole point is to find them again from wherever the application
// was started.
func recentListPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "simdiag", "recent.json")
}

// loadRecent returns the remembered configurations, most recent first.
//
// Nothing here is fatal: a missing, unreadable or corrupt list simply means the
// user has no shortcuts yet, which is also the state of a first run.
func loadRecent() []recentEntry {
	entries := make([]recentEntry, 0, recentLimit)

	for _, path := range readRecentPaths() {
		entries = append(entries, recentEntry{
			Path:    path,
			Name:    filepath.Base(path),
			Dir:     filepath.Dir(path),
			Missing: !fileExists(path),
		})
	}

	return entries
}

// mostRecentExisting returns the last configuration actually opened, skipping
// entries whose file has since been deleted or moved, or "" if there is none.
func mostRecentExisting() string {
	for _, path := range readRecentPaths() {
		if fileExists(path) {
			return path
		}
	}
	return ""
}

// rememberRecent moves a configuration to the top of the list.
//
// Best effort by design: failing to write a shortcut list must never stop the
// user from working on the configuration they just opened.
func rememberRecent(path string) {
	listPath := recentListPath()
	if listPath == "" {
		return
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}

	updated := append([]string{filepath.Clean(abs)}, readRecentPaths()...)
	updated = dedupePaths(updated)
	if len(updated) > recentLimit {
		updated = updated[:recentLimit]
	}

	data, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(listPath), 0755); err != nil {
		return
	}
	_ = os.WriteFile(listPath, data, 0644)
}

// readRecentPaths reads the stored paths, most recent first.
func readRecentPaths() []string {
	listPath := recentListPath()
	if listPath == "" {
		return nil
	}

	data, err := os.ReadFile(listPath)
	if err != nil {
		return nil
	}

	var paths []string
	if err := json.Unmarshal(data, &paths); err != nil {
		return nil
	}

	return dedupePaths(paths)
}

// dedupePaths keeps the first occurrence of each path, dropping empty ones.
// Comparison is case-insensitive: this is a Windows application, and
// D:\sim\config.yaml and d:\sim\CONFIG.yaml are the same file.
func dedupePaths(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	unique := make([]string, 0, len(paths))

	for _, path := range paths {
		if path == "" {
			continue
		}
		clean := filepath.Clean(path)
		key := strings.ToLower(clean)
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, clean)
	}

	return unique
}
