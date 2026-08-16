package update

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The backup belongs next to the executable, and this is the regression that
// matters: the GUI chdirs into the configuration's directory at startup
// (gui/state.go, enterConfigDirectory), so a working-directory-relative name
// would move the running executable into the user's profile folder, and fail
// outright when the two are on different volumes, since os.Rename cannot cross
// them on Windows.
func TestBackupPath_SitsNextToTheExecutable(t *testing.T) {
	t.Chdir(t.TempDir())

	got, err := backupPath()
	if err != nil {
		t.Fatalf("backupPath: %v", err)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	if want := filepath.Join(filepath.Dir(exe), backupName); got != want {
		t.Errorf("backupPath = %q, want %q", got, want)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("backupPath = %q, want an absolute path", got)
	}
}

func TestReplaceBinary_SwapsInPlaceAndKeepsTheBackupAlongside(t *testing.T) {
	installDir := t.TempDir()
	currentExe := filepath.Join(installDir, executableName)
	if err := os.WriteFile(currentExe, []byte("old binary"), 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	staging := t.TempDir()
	if err := os.WriteFile(filepath.Join(staging, executableName), []byte("new binary"), 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Somewhere else entirely, the way the GUI runs.
	t.Chdir(t.TempDir())

	if err := replaceBinary(staging, currentExe); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	installed, err := os.ReadFile(currentExe)
	if err != nil {
		t.Fatalf("reading the installed binary: %v", err)
	}
	if string(installed) != "new binary" {
		t.Errorf("installed binary = %q, want the new one", installed)
	}

	backup, err := os.ReadFile(filepath.Join(installDir, backupName))
	if err != nil {
		t.Fatalf("the backup is not beside the executable: %v", err)
	}
	if string(backup) != "old binary" {
		t.Errorf("backup = %q, want the old binary", backup)
	}
}

func TestValidateZip(t *testing.T) {
	t.Run("accepts the executable at the root", func(t *testing.T) {
		if err := validateZip(writeZip(t, executableName)); err != nil {
			t.Errorf("validateZip: %v", err)
		}
	})

	// validateZip and replaceBinary have to agree on where the executable is:
	// accepting a nested layout here would pass validation and then fail at
	// install, after the running binary had already been renamed.
	t.Run("refuses it nested in a directory", func(t *testing.T) {
		if err := validateZip(writeZip(t, "simdiag_0.4.0/"+executableName)); err == nil {
			t.Error("validateZip accepted a nested executable")
		}
	})

	t.Run("refuses an archive without it", func(t *testing.T) {
		if err := validateZip(writeZip(t, "README.md")); err == nil {
			t.Error("validateZip accepted an archive with no executable")
		}
	})
}

func TestExtractFile_RefusesPathsEscapingTheStagingDirectory(t *testing.T) {
	zipPath := writeZip(t, "../escaped.exe")

	if err := extractZip(zipPath, t.TempDir()); err == nil {
		t.Error("extractZip accepted a path outside the staging directory")
	} else if !strings.Contains(err.Error(), "illegal file path") {
		t.Errorf("extractZip error = %v, want it to name the illegal path", err)
	}
}

// writeZip builds a one-entry archive and returns its path.
func writeZip(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "archive.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer file.Close()

	w := zip.NewWriter(file)
	entry, err := w.Create(name)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := entry.Write([]byte("content")); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("setup: %v", err)
	}

	return path
}
