// Package update keeps SimDiag current from its GitHub releases.
//
// It is split in three so both front ends can use it. The CLI's
// simdiag.exe update and the GUI's About tab both go through LatestRelease,
// Compare and Apply, rather than each having their own idea of what a newer
// version is.
package update

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"simdiag/common"
)

const (
	executableName = "simdiag.exe"

	// backupName is the running executable, moved aside. Windows allows renaming
	// a running image but not overwriting it, which is the whole trick.
	backupName = "simdiag.old.exe"
)

// Run is the command line entry point: simdiag.exe update.
func Run() error {
	ctx := context.Background()

	common.Printf("SimDiag Self-Update\n")
	common.Printf("Current version: %s\n\n", common.SimdiagVersion)

	CleanupBackup()

	common.Printf("Fetching latest release...\n")
	release, err := LatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}

	common.Printf("Latest version: %s\n\n", release.Version)

	if !IsDevelopmentBuild(common.SimdiagVersion) && Compare(common.SimdiagVersion, release.Version) >= 0 {
		common.Printf("Already up to date!\n")
		return nil
	}

	if _, err := Apply(ctx, release); err != nil {
		return err
	}

	common.Printf("\nUpdate complete! Run './simdiag.exe -v' to verify.\n")
	return nil
}

// Apply installs a release over the running executable and returns the path it
// was installed at.
//
// That path is captured before the swap on purpose: once the running image has
// been renamed, os.Executable can report the backup's name instead, and the
// caller (the GUI's restart button) needs the path to launch.
//
// Progress goes through common.Printf, never fmt: the GUI captures a run by
// redirecting common.SetOutput, while a fmt call lands on a standard output the
// -H windowsgui binary does not have and is lost without trace.
func Apply(ctx context.Context, release *Release) (string, error) {
	currentExe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to locate the current executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(currentExe); err == nil {
		currentExe = resolved
	}

	common.Printf("Downloading %s...\n", release.AssetName)
	zipPath, sum, err := downloadAsset(ctx, release)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(zipPath)

	common.Printf("Verifying checksum...\n")
	expected, err := fetchChecksum(ctx, release.ChecksumURL, release.AssetName)
	if err != nil {
		return "", err
	}
	if sum != expected {
		return "", fmt.Errorf("checksum mismatch for %s: expected %s, got %s: the download was not installed",
			release.AssetName, expected, sum)
	}
	common.Printf("  sha256 %s\n", sum)

	if err := validateZip(zipPath); err != nil {
		return "", fmt.Errorf("invalid update package: %w", err)
	}

	stagingDir, err := os.MkdirTemp("", "simdiag-update-*")
	if err != nil {
		return "", fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	if err := extractZip(zipPath, stagingDir); err != nil {
		return "", fmt.Errorf("failed to extract update: %w", err)
	}

	common.Printf("\nUpdating binary...\n")
	if err := replaceBinary(stagingDir, currentExe); err != nil {
		return "", fmt.Errorf("failed to update binary: %w", err)
	}

	common.Printf("  simdiag.exe updated (%s -> %s)\n", common.SimdiagVersion, release.Version)

	// The base templates travel inside the binary and are written to disk by the
	// graphical interface, so an update has nothing to copy: whatever is on disk
	// belongs to the user, and the new exe carries its own copies.
	return currentExe, nil
}

// CleanupBackup removes the executable a previous update moved aside.
//
// It is resolved next to the running executable, never against the working
// directory: the GUI chdirs into the configuration's folder at startup
// (gui/state.go, enterConfigDirectory), so a relative name would look for the
// backup in the user's profile directory, and leave the real one behind.
func CleanupBackup() {
	path, err := backupPath()
	if err != nil {
		return
	}
	if _, err := os.Stat(path); err == nil {
		os.Remove(path) // Best effort: it is only leftover.
	}
}

// backupPath is where the running executable is moved aside.
func backupPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Join(filepath.Dir(exe), backupName), nil
}

// downloadAsset fetches the release archive, returning its path and the sha256
// computed as it was written. Verifying costs no second read of the file.
func downloadAsset(ctx context.Context, release *Release) (string, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, release.AssetURL, nil)
	if err != nil {
		return "", "", err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download failed with status: %s", response.Status)
	}

	tmpFile, err := os.CreateTemp("", "simdiag-download-*.zip")
	if err != nil {
		return "", "", err
	}
	defer tmpFile.Close()

	digest := sha256.New()
	if err := copyWithProgress(io.MultiWriter(tmpFile, digest), response.Body, response.ContentLength); err != nil {
		os.Remove(tmpFile.Name())
		return "", "", err
	}

	return tmpFile.Name(), hex.EncodeToString(digest.Sum(nil)), nil
}

// copyWithProgress copies the body, reporting every 10% when the size is known.
func copyWithProgress(dst io.Writer, src io.Reader, size int64) error {
	buf := make([]byte, 32*1024)
	written := int64(0)
	lastPercent := -1

	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			if _, err := dst.Write(buf[:n]); err != nil {
				return err
			}
			written += int64(n)

			if size > 0 {
				percent := int(written * 100 / size)
				if percent != lastPercent && percent%10 == 0 {
					common.Printf("  %d%%\n", percent)
					lastPercent = percent
				}
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

// validateZip ensures the archive holds the executable where replaceBinary
// looks for it: at the archive root, which is the flat layout goreleaser
// produces (archives has no wrap_in_directory).
func validateZip(zipPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == executableName {
			return nil
		}
	}

	return fmt.Errorf("archive does not contain %s at its root", executableName)
}

// extractZip extracts all files from the zip to the staging directory.
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if err := extractFile(f, destDir); err != nil {
			return err
		}
	}

	return nil
}

// extractFile extracts a single file from the zip.
func extractFile(f *zip.File, destDir string) error {
	path := filepath.Join(destDir, f.Name)

	// Check for ZipSlip vulnerability
	if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, os.ModePerm)
	}

	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}

	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

// replaceBinary puts the new executable at currentExe, keeping the old one
// alongside until the next run so a failure can be rolled back.
func replaceBinary(stagingDir, currentExe string) error {
	newExePath := filepath.Join(stagingDir, executableName)
	if _, err := os.Stat(newExePath); os.IsNotExist(err) {
		return fmt.Errorf("%s not found in extracted files", executableName)
	}

	oldExe := filepath.Join(filepath.Dir(currentExe), backupName)

	// Windows allows renaming a running image, which is what makes replacing the
	// executable in place possible. Keeping the backup in the same directory also
	// keeps the rename on one volume: os.Rename cannot cross volumes.
	if err := os.Rename(currentExe, oldExe); err != nil {
		return fmt.Errorf("failed to rename current executable (try running as administrator): %w", err)
	}

	if err := copyFile(newExePath, currentExe); err != nil {
		if renameErr := os.Rename(oldExe, currentExe); renameErr != nil {
			return fmt.Errorf("failed to copy new executable and rollback failed: %w (rollback error: %v)", err, renameErr)
		}
		return fmt.Errorf("failed to copy new executable: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}
