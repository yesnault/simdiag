package update

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// An update replaces the executable the user is running, and this is the only
// moment at which what was downloaded can still be checked. goreleaser has been
// publishing checksums.txt alongside every archive all along; nothing read it.
//
// The file is plain sha256sum output, one line per asset:
//
//	e6870de26a288fd8e1d5dfd8464305cf90ed188baa4e4f842bfd0be487ffe926  simdiag_0.3.0_windows_amd64.zip

// fetchChecksum returns the expected sha256 of assetName, in lower case hex.
//
// Every failure is fatal to the update, deliberately: a missing checksums.txt,
// a file that does not mention this asset, or an unreadable one all stop the
// install. A verification that is skipped when the file is absent verifies
// nothing. It only moves the hole.
func fetchChecksum(ctx context.Context, checksumURL, assetName string) (string, error) {
	if checksumURL == "" {
		return "", fmt.Errorf("the release publishes no %s, so the download cannot be verified", checksumsAssetName)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return "", err
	}

	response, err := (&http.Client{Timeout: requestTimeout}).Do(request)
	if err != nil {
		return "", fmt.Errorf("unable to download %s: %w", checksumsAssetName, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unable to download %s: %s", checksumsAssetName, response.Status)
	}

	return parseChecksums(response.Body, assetName)
}

// parseChecksums finds the line covering assetName.
func parseChecksums(r io.Reader, assetName string) (string, error) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		// "<hex>  <name>", with the name possibly marked binary as "*<name>".
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			continue
		}

		sum, name := fields[0], strings.TrimPrefix(fields[1], "*")
		if name != assetName {
			continue
		}

		if len(sum) != 64 {
			return "", fmt.Errorf("%s lists a malformed checksum for %s", checksumsAssetName, assetName)
		}
		return strings.ToLower(sum), nil
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("unable to read %s: %w", checksumsAssetName, err)
	}

	return "", fmt.Errorf("%s does not cover %s", checksumsAssetName, assetName)
}
