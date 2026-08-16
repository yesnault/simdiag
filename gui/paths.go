package gui

import (
	"os"
	"path/filepath"
)

const configFileName = "mapping_config.yaml"

// DefaultConfigPath decides where mapping_config.yaml lives for a windowed app.
//
// The CLI resolves the config against the working directory, which is fine when
// it is launched from the install folder. A GUI started from the Start menu or a
// shortcut inherits an arbitrary working directory, so the path has to be
// derived from the executable instead.
//
// This is only the first-run answer: once a configuration has been opened, Run
// reopens the last one used (see mostRecentExisting), and -c always wins.
//
// Preference order: an existing config next to the executable (how current
// SimDiag users are set up), then an existing one under %APPDATA%, and for a
// first run, next to the executable when that directory is writable, falling
// back to %APPDATA% for installs under Program Files.
func DefaultConfigPath() string {
	nextToExe := filepath.Join(exeDir(), configFileName)
	if exeDir() != "" && fileExists(nextToExe) {
		return nextToExe
	}

	inAppData := appDataConfigPath()
	if inAppData != "" && fileExists(inAppData) {
		return inAppData
	}

	if exeDir() != "" && dirWritable(exeDir()) {
		return nextToExe
	}
	if inAppData != "" {
		return inAppData
	}
	return configFileName
}

// exeDir returns the directory holding the running executable, or "".
func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe)
}

// appDataConfigPath returns %APPDATA%\simdiag\mapping_config.yaml, or "".
func appDataConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "simdiag", configFileName)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// dirWritable reports whether a file can be created in dir. Program Files is
// read-only for a standard user, and that is exactly the case we must detect.
func dirWritable(dir string) bool {
	probe, err := os.CreateTemp(dir, ".simdiag-write-probe-*")
	if err != nil {
		return false
	}
	name := probe.Name()
	probe.Close()
	os.Remove(name)
	return true
}
