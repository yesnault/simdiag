package update

import (
	"strings"
	"testing"
)

// The real checksums.txt of v0.3.0, as GitHub serves it.
const realChecksums = "e6870de26a288fd8e1d5dfd8464305cf90ed188baa4e4f842bfd0be487ffe926  simdiag_0.3.0_windows_amd64.zip\n"

func TestParseChecksums_FindsTheAssetLine(t *testing.T) {
	got, err := parseChecksums(strings.NewReader(realChecksums), "simdiag_0.3.0_windows_amd64.zip")
	if err != nil {
		t.Fatalf("parseChecksums: %v", err)
	}

	const want = "e6870de26a288fd8e1d5dfd8464305cf90ed188baa4e4f842bfd0be487ffe926"
	if got != want {
		t.Errorf("parseChecksums = %q, want %q", got, want)
	}
}

// A release with several assets must not hand back a neighbour's checksum.
func TestParseChecksums_PicksTheRightLineAmongSeveral(t *testing.T) {
	file := "1111111111111111111111111111111111111111111111111111111111111111  simdiag_0.4.0_linux_amd64.zip\n" +
		"2222222222222222222222222222222222222222222222222222222222222222  simdiag_0.4.0_windows_amd64.zip\n" +
		"3333333333333333333333333333333333333333333333333333333333333333  simdiag_0.4.0_windows_arm64.zip\n"

	got, err := parseChecksums(strings.NewReader(file), "simdiag_0.4.0_windows_amd64.zip")
	if err != nil {
		t.Fatalf("parseChecksums: %v", err)
	}
	if got != strings.Repeat("2", 64) {
		t.Errorf("parseChecksums = %q, want the windows_amd64 line", got)
	}
}

// sha256sum marks binary files with a leading star, and an uppercase digest is
// still the same digest.
func TestParseChecksums_AcceptsTheBinaryMarkerAndUpperCase(t *testing.T) {
	file := strings.ToUpper(strings.Repeat("a", 64)) + " *simdiag_0.4.0_windows_amd64.zip\n"

	got, err := parseChecksums(strings.NewReader(file), "simdiag_0.4.0_windows_amd64.zip")
	if err != nil {
		t.Fatalf("parseChecksums: %v", err)
	}
	if got != strings.Repeat("a", 64) {
		t.Errorf("parseChecksums = %q, want it lower-cased", got)
	}
}

// Every way the file can fail to cover the asset has to be an error. A check
// that passes when the answer is missing does not check anything. It only moves
// the hole somewhere less visible.
func TestParseChecksums_RefusesAnythingItCannotVouchFor(t *testing.T) {
	for name, file := range map[string]string{
		"empty file":         "",
		"asset not listed":   "1111111111111111111111111111111111111111111111111111111111111111  something_else.zip\n",
		"truncated checksum": "abc123  simdiag_0.4.0_windows_amd64.zip\n",
		"no checksum column": "simdiag_0.4.0_windows_amd64.zip\n",
	} {
		if got, err := parseChecksums(strings.NewReader(file), "simdiag_0.4.0_windows_amd64.zip"); err == nil {
			t.Errorf("%s: parseChecksums = %q, want an error", name, got)
		}
	}
}

// A release that publishes no checksums.txt cannot be installed.
func TestFetchChecksum_RefusesAReleaseWithoutChecksums(t *testing.T) {
	if _, err := fetchChecksum(t.Context(), "", "simdiag_0.4.0_windows_amd64.zip"); err == nil {
		t.Error("fetchChecksum with no checksum URL returned no error")
	}
}
