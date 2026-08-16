package update

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v73/github"
)

const (
	releaseOwner = "yesnault"
	releaseRepo  = "simdiag"

	// The only build published: .goreleaser.yml names the archive
	// simdiag_<version>_windows_amd64.zip.
	windowsAssetSuffix = "windows_amd64.zip"

	// checksumsAssetName is goreleaser's checksum.name_template. Without it an
	// update cannot be verified, and an unverified update is not installed.
	checksumsAssetName = "checksums.txt"

	// GitHub has no reason to take longer, and http.DefaultClient has no timeout
	// at all, and a stalled handshake would hold a GUI goroutine open for good.
	requestTimeout = 30 * time.Second
)

// Release is what the interface shows about a version.
//
// It carries no go-github type, which is what stops that dependency at this
// package's edge: the GUI talks about releases without knowing where they come
// from.
type Release struct {
	Version     string    // "0.4.0", the tag without its leading v
	Notes       string    // the release body, Markdown as GitHub stores it
	URL         string    // the release page
	PublishedAt time.Time // zero when GitHub reports none
	AssetURL    string
	AssetName   string
	AssetSize   int64
	ChecksumURL string
}

// LatestRelease reports the newest published release.
//
// GitHub's "latest" excludes drafts and pre-releases by design, so nothing here
// has to filter them out.
func LatestRelease(ctx context.Context) (*Release, error) {
	client := github.NewClient(&http.Client{Timeout: requestTimeout})

	release, response, err := client.Repositories.GetLatestRelease(ctx, releaseOwner, releaseRepo)
	if err != nil {
		if response != nil && response.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("GitHub API rate limit exceeded (60/hour without authentication)")
		}
		return nil, err
	}

	asset, err := findWindowsAsset(release)
	if err != nil {
		return nil, err
	}

	return &Release{
		Version:     strings.TrimPrefix(release.GetTagName(), "v"),
		Notes:       strings.TrimSpace(release.GetBody()),
		URL:         release.GetHTMLURL(),
		PublishedAt: release.GetPublishedAt().Time,
		AssetURL:    asset.GetBrowserDownloadURL(),
		AssetName:   asset.GetName(),
		AssetSize:   int64(asset.GetSize()),
		ChecksumURL: findAssetURL(release, checksumsAssetName),
	}, nil
}

// findWindowsAsset picks the archive this build can install.
func findWindowsAsset(release *github.RepositoryRelease) (*github.ReleaseAsset, error) {
	for _, asset := range release.Assets {
		if strings.Contains(asset.GetName(), windowsAssetSuffix) {
			return asset, nil
		}
	}

	names := make([]string, 0, len(release.Assets))
	for _, asset := range release.Assets {
		names = append(names, asset.GetName())
	}

	return nil, fmt.Errorf("no Windows amd64 asset in release %s (found: %s)",
		release.GetTagName(), strings.Join(names, ", "))
}

// findAssetURL returns the download URL of a named asset, or "".
func findAssetURL(release *github.RepositoryRelease, name string) string {
	for _, asset := range release.Assets {
		if asset.GetName() == name {
			return asset.GetBrowserDownloadURL()
		}
	}
	return ""
}
