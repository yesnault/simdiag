package dcs

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// inputSubdir is where DCS keeps one folder per aircraft, plus a few profiles
// that are not aircraft at all.
func inputSubdir(basePath string) string {
	return filepath.Join(basePath, "Config", "Input")
}

// The three rules below decide what Config/Input contains. parseDCS and
// ListModules each apply the subset that concerns them, but each rule is written
// once so the two cannot end up disagreeing about what a module is.

// systemProfiles are the folders that exist for every install and belong to no
// aircraft. They are still parsed, since their bindings are shared across modules,
// but they are not modules, so they get no diagram directory of their own.
var systemProfiles = map[string]bool{
	"Default":     true,
	"UiLayer":     true,
	"CommandMenu": true,
}

// isEasyVariant reports the simplified control scheme DCS writes alongside a real
// profile. Parsing it would export the same aircraft twice.
func isEasyVariant(name string) bool {
	return strings.Contains(name, "_easy")
}

// hasJoystickFolder reports whether a profile binds any controller at all. A
// profile without one has nothing to draw.
func hasJoystickFolder(profilePath string) bool {
	info, err := os.Stat(filepath.Join(profilePath, "joystick"))
	return err == nil && info.IsDir()
}

// ListModules returns the aircraft found under a DCS installation's Config/Input,
// sorted, without parsing a single Lua file.
//
// Both front ends show this list as soon as the user picks the DCS path, so it
// has to be cheap: the full parseDCS reads every .diff.lua of every profile,
// which takes seconds, while this is one ReadDir and a stat per folder. The two
// stay in agreement by sharing the three rules above.
func ListModules(basePath string) ([]string, error) {
	inputPath := inputSubdir(basePath)

	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return nil, fmt.Errorf("error reading Input folder %s: %w", inputPath, err)
	}

	modules := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || isEasyVariant(name) || systemProfiles[name] {
			continue
		}
		if !hasJoystickFolder(filepath.Join(inputPath, name)) {
			continue
		}
		modules = append(modules, name)
	}

	slices.Sort(modules)
	return modules, nil
}
