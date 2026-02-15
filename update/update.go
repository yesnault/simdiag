package update

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"simdiag/common"

	"github.com/google/go-github/v69/github"
)

// Run executes the self-update process
func Run() error {
	fmt.Println("SimDiag Self-Update")
	fmt.Printf("Current version: %s\n", common.SimdiagVersion)
	fmt.Println()

	// Step 1: Cleanup old exe if exists
	cleanupOldExecutable()

	// Step 2: Fetch latest release from GitHub
	fmt.Println("Fetching latest release...")
	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.GetTagName(), "v")
	fmt.Printf("Latest version: %s\n", latestVersion)
	fmt.Println()

	// Step 3: Compare versions
	if common.SimdiagVersion != "dev" && common.SimdiagVersion == latestVersion {
		fmt.Println("Already up to date!")
		return nil
	}

	// Step 4: Find Windows amd64 asset
	asset, err := findWindowsAsset(release)
	if err != nil {
		return err
	}

	// Step 5: Download the zip file
	zipPath, err := downloadAsset(asset)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(zipPath) // Cleanup downloaded zip

	// Step 6: Validate the zip
	if err := validateZip(zipPath); err != nil {
		return fmt.Errorf("invalid update package: %w", err)
	}

	// Step 7: Extract to staging directory
	stagingDir, err := os.MkdirTemp("", "simdiag-update-*")
	if err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir) // Cleanup staging dir

	if err := extractZip(zipPath, stagingDir); err != nil {
		return fmt.Errorf("failed to extract update: %w", err)
	}

	// Step 8: Replace the binary
	fmt.Println("\nUpdating binary...")
	if err := replaceBinary(stagingDir, common.SimdiagVersion, latestVersion); err != nil {
		return fmt.Errorf("failed to update binary: %w", err)
	}

	// Step 9: Update templates
	fmt.Println("\nUpdating templates...")
	if err := updateTemplates(stagingDir); err != nil {
		return fmt.Errorf("failed to update templates: %w", err)
	}

	fmt.Println("\nUpdate complete! Run './simdiag.exe -v' to verify.")
	return nil
}

// cleanupOldExecutable removes any leftover .old.exe file from previous updates
func cleanupOldExecutable() {
	oldExe := "simdiag.old.exe"
	if _, err := os.Stat(oldExe); err == nil {
		os.Remove(oldExe) // Best effort, ignore errors
	}
}

// fetchLatestRelease fetches the latest release from GitHub
func fetchLatestRelease() (*github.RepositoryRelease, error) {
	client := github.NewClient(nil)
	ctx := context.Background()

	release, resp, err := client.Repositories.GetLatestRelease(ctx, "yesnault", "simdiag")
	if err != nil {
		// Check for rate limit
		if resp != nil && resp.StatusCode == 403 {
			return nil, fmt.Errorf("GitHub API rate limit exceeded (60/hour without auth)")
		}
		return nil, err
	}

	return release, nil
}

// findWindowsAsset finds the Windows amd64 zip asset in the release
func findWindowsAsset(release *github.RepositoryRelease) (*github.ReleaseAsset, error) {
	for _, asset := range release.Assets {
		name := asset.GetName()
		if strings.Contains(name, "windows_amd64.zip") {
			return asset, nil
		}
	}

	// No Windows asset found - list available assets
	var assetNames []string
	for _, asset := range release.Assets {
		assetNames = append(assetNames, asset.GetName())
	}

	return nil, fmt.Errorf("no Windows amd64 asset found in release\nAvailable assets: %v", assetNames)
}

// downloadAsset downloads a release asset to a temp file with progress indication
func downloadAsset(asset *github.ReleaseAsset) (string, error) {
	url := asset.GetBrowserDownloadURL()
	name := asset.GetName()

	fmt.Printf("Downloading %s...\n", name)

	// Create temp file
	tmpFile, err := os.CreateTemp("", "simdiag-download-*.zip")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Download
	resp, err := http.Get(url)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Copy with simple progress
	size := resp.ContentLength
	downloaded := int64(0)
	lastPercent := -1

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := tmpFile.Write(buf[:n]); writeErr != nil {
				os.Remove(tmpFile.Name())
				return "", writeErr
			}
			downloaded += int64(n)

			// Show progress
			if size > 0 {
				percent := int(downloaded * 100 / size)
				if percent != lastPercent && percent%10 == 0 {
					fmt.Printf("  %d%%\n", percent)
					lastPercent = percent
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(tmpFile.Name())
			return "", err
		}
	}

	fmt.Println("  100%")
	return tmpFile.Name(), nil
}

// validateZip ensures the zip contains simdiag.exe
func validateZip(zipPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "simdiag.exe") {
			return nil
		}
	}

	return fmt.Errorf("archive does not contain simdiag.exe")
}

// extractZip extracts all files from the zip to the staging directory
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

// extractFile extracts a single file from the zip
func extractFile(f *zip.File, destDir string) error {
	path := filepath.Join(destDir, f.Name)

	// Check for ZipSlip vulnerability
	if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, os.ModePerm)
	}

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}

	// Extract file
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

// replaceBinary replaces the current executable with the new one
func replaceBinary(stagingDir, oldVersion, newVersion string) error {
	// Find the new exe in staging
	newExePath := filepath.Join(stagingDir, "simdiag.exe")
	if _, err := os.Stat(newExePath); os.IsNotExist(err) {
		return fmt.Errorf("simdiag.exe not found in extracted files")
	}

	// Get current exe path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	oldExe := "simdiag.old.exe"

	// Rename current exe to .old (Windows allows this even for running exe)
	if err := os.Rename(currentExe, oldExe); err != nil {
		return fmt.Errorf("failed to rename current executable (try running as administrator): %w", err)
	}

	// Copy new exe to current location
	if err := copyFile(newExePath, currentExe); err != nil {
		// Rollback: restore old exe
		if renameErr := os.Rename(oldExe, currentExe); renameErr != nil {
			return fmt.Errorf("failed to copy new executable and rollback failed: %w (rollback error: %v)", err, renameErr)
		}
		return fmt.Errorf("failed to copy new executable: %w", err)
	}

	fmt.Printf("  simdiag.exe updated (%s -> %s)\n", oldVersion, newVersion)
	return nil
}

// updateTemplates extracts templates with conflict resolution
func updateTemplates(stagingDir string) error {
	templatesDir := "./templates"
	stagingTemplatesDir := filepath.Join(stagingDir, "templates")

	// Check if staging has templates
	if _, err := os.Stat(stagingTemplatesDir); os.IsNotExist(err) {
		fmt.Println("  No templates in update package")
		return nil
	}

	// Create local templates directory if it doesn't exist
	if err := os.MkdirAll(templatesDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Walk through staging templates
	return filepath.Walk(stagingTemplatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(stagingTemplatesDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(templatesDir, relPath)

		// Check if file already exists
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			// New file - copy silently
			if err := copyFile(path, destPath); err != nil {
				return err
			}
			fmt.Printf("  [new] %s\n", relPath)
		} else {
			// Existing file - prompt user
			fmt.Printf("  [exists] %s - Overwrite? (y/n): ", relPath)
			var response string
			if _, scanErr := fmt.Scanln(&response); scanErr != nil {
				// Treat scan error as "no" (keep local version)
				fmt.Println("    → kept local version")
				return nil
			}

			if strings.ToLower(strings.TrimSpace(response)) == "y" {
				if err := copyFile(path, destPath); err != nil {
					return err
				}
				fmt.Println("    → updated")
			} else {
				fmt.Println("    → kept local version")
			}
		}

		return nil
	})
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// Create parent directory if needed
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

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}
