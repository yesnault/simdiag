// Package templates carries the diagrams SimDiag ships with: the controllers
// common enough that every user wants them.
//
// They are embedded so the binary travels alone (the README tells people to
// download simdiag.exe and nothing else) but they are written to disk rather
// than read from the binary, because a template is meant to be opened, edited
// and added to. What is on disk always wins.
//
// This directory is both a Go package and the repository's own templates
// directory. go:embed cannot reach outside its package, and the alternative was
// a second copy of 3.8 MB of SVG. The pattern below does not descend into
// subdirectories, which is what keeps the bespoke button boxes in
// yesnault_custom/ out of the binary: they are one pilot's hardware, of no use
// to anyone else.
package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

//go:embed *.svg
var baseFS embed.FS

// Names returns the base templates, sorted.
func Names() []string {
	entries, err := fs.ReadDir(baseFS, ".")
	if err != nil {
		// Unreachable: the filesystem is compiled in.
		return nil
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	slices.Sort(names)

	return names
}

// Read returns one base template's contents.
func Read(name string) ([]byte, error) {
	return baseFS.ReadFile(name)
}

// Missing returns the base templates that are not in dir, sorted.
//
// A directory that does not exist yet simply misses all of them, which is the
// first run, not an error.
func Missing(dir string) []string {
	if dir == "" {
		return Names()
	}

	var missing []string
	for _, name := range Names() {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			missing = append(missing, name)
		}
	}

	return missing
}

// Install writes the base templates that dir does not already have, creating it
// if needed, and returns what it wrote.
//
// An existing file is never touched, whatever its contents: a template someone
// has edited is their work, and there is no way to tell an edit from an
// untouched copy that matters enough to risk it.
func Install(dir string) ([]string, error) {
	if dir == "" {
		return nil, fmt.Errorf("no templates directory to install into")
	}

	missing := Missing(dir)
	if len(missing) == 0 {
		return nil, nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("unable to create %s: %w", dir, err)
	}

	written := make([]string, 0, len(missing))
	for _, name := range missing {
		data, err := Read(name)
		if err != nil {
			return written, fmt.Errorf("unable to read the embedded %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			return written, fmt.Errorf("unable to write %s: %w", name, err)
		}
		written = append(written, name)
	}

	return written, nil
}
