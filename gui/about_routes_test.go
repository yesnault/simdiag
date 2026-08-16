package gui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"simdiag/common"
	"simdiag/update"
)

// stubLatest replaces the GitHub call for the length of a test, and points the
// settings store at a temporary directory so the six-hour cache does not leak
// between tests or touch the user's real preferences.
func stubLatest(t *testing.T, release *update.Release, err error) {
	t.Helper()

	t.Setenv("APPDATA", t.TempDir())

	previous := fetchLatest
	fetchLatest = func(context.Context) (*update.Release, error) { return release, err }
	t.Cleanup(func() { fetchLatest = previous })
}

func releaseFixture(version string) *update.Release {
	return &update.Release{
		Version:     version,
		Notes:       "## What's new\n- an About tab",
		URL:         "https://github.com/yesnault/simdiag/releases/tag/v" + version,
		PublishedAt: time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC),
		AssetName:   "simdiag_" + version + "_windows_amd64.zip",
	}
}

func getCheck(t *testing.T, version, query string) updateCheck {
	t.Helper()

	state := newTestState(t, &common.Config{})
	state.version = version

	rec := httptest.NewRecorder()
	NewHandler(state).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/update/check"+query, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/update/check = %d: %s", rec.Code, rec.Body.String())
	}

	var got updateCheck
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return got
}

func TestUpdateCheck_ReportsANewerRelease(t *testing.T) {
	stubLatest(t, releaseFixture("0.4.0"), nil)

	got := getCheck(t, "0.3.0", "")
	if !got.Available {
		t.Errorf("available = false, want true for 0.3.0 against 0.4.0")
	}
	if got.Latest == nil || got.Latest.Version != "0.4.0" {
		t.Fatalf("latest = %+v, want 0.4.0", got.Latest)
	}
	if !strings.Contains(got.Latest.Notes, "About tab") {
		t.Errorf("notes = %q, want the release body", got.Latest.Notes)
	}
	if got.Latest.PublishedAt == "" {
		t.Error("publishedAt is empty, so the page cannot date the release")
	}
}

func TestUpdateCheck_ReportsNothingWhenCurrent(t *testing.T) {
	stubLatest(t, releaseFixture("0.4.0"), nil)

	if got := getCheck(t, "0.4.0", ""); got.Available {
		t.Errorf("available = true while running the latest version")
	}
}

// A snapshot is newer than the release it follows. This is the case the old
// string comparison got backwards, and with a one-click button behind it the
// answer must be "up to date", never an offer to install 0.3.0 over 0.3.1-next.
func TestUpdateCheck_DoesNotOfferToDowngradeASnapshot(t *testing.T) {
	stubLatest(t, releaseFixture("0.3.0"), nil)

	if got := getCheck(t, "0.3.1-next", ""); got.Available {
		t.Error("available = true for 0.3.1-next against 0.3.0, which would downgrade")
	}
}

// go run reports "dev", which no release can be ordered against.
func TestUpdateCheck_OffersNothingToADevelopmentBuild(t *testing.T) {
	stubLatest(t, releaseFixture("0.4.0"), nil)

	got := getCheck(t, "dev", "")
	if !got.Development {
		t.Error("development = false for a dev build")
	}
	if got.Available {
		t.Error("available = true for a dev build")
	}
}

// Being offline is not a failure of anything the user asked for: the check runs
// unasked at startup, so it must come back 200 with the problem inside.
func TestUpdateCheck_ReportsAnUnreachableGitHubInThePayload(t *testing.T) {
	stubLatest(t, nil, fmt.Errorf("dial tcp: no such host"))

	got := getCheck(t, "0.3.0", "")
	if got.CheckFailed == "" {
		t.Error("checkFailed is empty, so the page cannot tell the user why nothing is shown")
	}
	if got.Available {
		t.Error("available = true despite the check having failed")
	}
}

// The remembered answer spares GitHub's 60 requests an hour, and ?force=1 is
// what the "Check for updates" button uses to bypass it.
func TestUpdateCheck_UsesTheRememberedAnswerUntilForced(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	calls := 0
	previous := fetchLatest
	fetchLatest = func(context.Context) (*update.Release, error) {
		calls++
		return releaseFixture("0.4.0"), nil
	}
	t.Cleanup(func() { fetchLatest = previous })

	getCheck(t, "0.3.0", "")
	if calls != 1 {
		t.Fatalf("first check made %d calls, want 1", calls)
	}

	if got := getCheck(t, "0.3.0", ""); !got.Available {
		t.Error("the remembered answer lost the fact that an update exists")
	}
	if calls != 1 {
		t.Errorf("second check made %d calls, want the remembered answer to be used", calls)
	}

	getCheck(t, "0.3.0", "?force=1")
	if calls != 2 {
		t.Errorf("forced check made %d calls in total, want 2", calls)
	}
}

// Replacing the binary underneath a running export is not something to allow.
func TestUpdateInstall_RefusesWhileAnExportRuns(t *testing.T) {
	stubLatest(t, releaseFixture("0.4.0"), nil)

	state := newTestState(t, &common.Config{})
	state.version = "0.3.0"

	if !currentExport.begin(func() {}) {
		t.Fatal("setup: could not claim the export slot")
	}
	t.Cleanup(func() { currentExport.finish(nil, "") })

	rec := httptest.NewRecorder()
	NewHandler(state).ServeHTTP(rec,
		httptest.NewRequest(http.MethodPost, "/api/update/install", strings.NewReader("{}")))

	if rec.Code != http.StatusConflict {
		t.Errorf("POST /api/update/install during an export = %d, want 409", rec.Code)
	}
}

func TestUpdateInstall_RefusesWhenAlreadyCurrent(t *testing.T) {
	stubLatest(t, releaseFixture("0.4.0"), nil)

	state := newTestState(t, &common.Config{})
	state.version = "0.4.0"

	rec := httptest.NewRecorder()
	NewHandler(state).ServeHTTP(rec,
		httptest.NewRequest(http.MethodPost, "/api/update/install", strings.NewReader("{}")))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /api/update/install on the latest version = %d, want 400", rec.Code)
	}
}

// Nothing installed means nothing to restart into, and the button must say so
// rather than launching whatever is at the executable's path.
func TestUpdateRestart_RefusesWithNothingInstalled(t *testing.T) {
	installedExePath.Lock()
	installedExePath.path = ""
	installedExePath.Unlock()

	rec := httptest.NewRecorder()
	NewHandler(newTestState(t, &common.Config{})).ServeHTTP(rec,
		httptest.NewRequest(http.MethodPost, "/api/update/restart", strings.NewReader("{}")))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /api/update/restart with nothing installed = %d, want 400", rec.Code)
	}
}

// The release link is the one destination whose URL is not known in advance, so
// it must not be openable before a check has established what it is.
func TestOpenURL_ReleaseTargetIsRefusedUntilAReleaseIsKnown(t *testing.T) {
	rememberReleaseLink("")

	h := NewHandler(newTestState(t, &common.Config{}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/open-url",
		strings.NewReader(`{"target":"release"}`)))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /api/open-url release with none known = %d, want 400", rec.Code)
	}
}
