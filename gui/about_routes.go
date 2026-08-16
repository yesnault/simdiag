package gui

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"simdiag/common"
	"simdiag/update"
)

// The About tab answers what an About screen answers: which version this is,
// whether a newer one exists and can be installed from here, where the project
// lives, under what licence. It is also the one place that reaches out to
// GitHub.

// checkInterval is how long a look at GitHub stays good for. The frontend asks
// on every startup so the tab can carry a badge; without a cache that would be a
// network round trip each launch, against an API that allows 60 an hour
// unauthenticated.
const checkInterval = 6 * time.Hour

// fetchLatest is a variable so the routes can be tested without a network.
var fetchLatest = update.LatestRelease

// currentUpdate is the update's own run slot.
//
// It reuses exportRun but deliberately not currentExport: that instance is
// already shared by the export and the diagram regeneration, so taking it would
// make the Generate tab answer 409 and would block switching configuration
// (allowSwitch) for the length of a download.
var currentUpdate = &exportRun{}

// installedExePath is the path Apply installed at, kept for the restart button.
// os.Executable can report the backup's name once the running image has been
// renamed, so the path has to be the one captured before the swap.
var installedExePath struct {
	sync.Mutex
	path string
}

// releasePayload is a release as the interface shows it.
type releasePayload struct {
	Version     string `json:"version"`
	Notes       string `json:"notes,omitempty"`
	URL         string `json:"url,omitempty"`
	PublishedAt string `json:"publishedAt,omitempty"` // RFC3339, rendered by the page
}

// updateCheck is the answer to "is there anything newer".
type updateCheck struct {
	Current string `json:"current"`
	// Development reports a build no release can be ordered against, which is
	// every go run. There is nothing meaningful to offer such a build.
	Development bool            `json:"development"`
	Available   bool            `json:"available"`
	Latest      *releasePayload `json:"latest,omitempty"`
	// CheckFailed carries a look at GitHub that did not work out. It is not an
	// HTTP error: being offline is not a failure of anything the user asked for.
	CheckFailed string `json:"checkFailed,omitempty"`
}

func registerAboutRoutes(mux *http.ServeMux, state *State) {
	mux.HandleFunc("GET /api/update/check", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, checkForUpdate(r.Context(), state.Version(), r.URL.Query().Get("force") == "1"))
	})

	mux.HandleFunc("POST /api/update/install", func(w http.ResponseWriter, r *http.Request) {
		// Replacing the binary underneath a running export is the kind of thing
		// nobody wants to debug afterwards.
		if currentExport.isRunning() {
			writeMessageError(w, http.StatusConflict, msgExportRunning, nil)
			return
		}

		release, err := fetchLatest(r.Context())
		if err != nil {
			writeMessageError(w, http.StatusBadGateway, msgUpdateCheckFailed, errorArg(err))
			return
		}
		if !updateAvailable(state.Version(), release.Version) {
			writeMessageError(w, http.StatusBadRequest, msgAlreadyCurrent, nil)
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		if !currentUpdate.begin(cancel) {
			cancel()
			writeMessageError(w, http.StatusConflict, msgUpdateRunning, nil)
			return
		}

		go runUpdate(ctx, cancel, currentUpdate, release)

		w.WriteHeader(http.StatusAccepted)
		writeJSON(w, currentUpdate.stateSince(0))
	})

	mux.HandleFunc("GET /api/update/state", func(w http.ResponseWriter, r *http.Request) {
		from, _ := strconv.Atoi(r.URL.Query().Get("from"))
		writeJSON(w, currentUpdate.stateSince(from))
	})

	mux.HandleFunc("POST /api/update/cancel", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]bool{"cancelling": currentUpdate.requestCancel()})
	})

	mux.HandleFunc("POST /api/update/restart", func(w http.ResponseWriter, r *http.Request) {
		// An empty body is legal here: the button carries no arguments. Requests
		// reaching the asset server do not always have one.
		var ignored struct{}
		if err := json.NewDecoder(r.Body).Decode(&ignored); err != nil && !errors.Is(err, io.EOF) {
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}

		installedExePath.Lock()
		path := installedExePath.path
		installedExePath.Unlock()

		if path == "" {
			writeMessageError(w, http.StatusBadRequest, msgNothingInstalled, nil)
			return
		}

		if err := restartApplication(path); err != nil {
			writeMessageError(w, http.StatusInternalServerError, msgRestartFailed, errorArg(err))
			return
		}

		writeJSON(w, map[string]bool{"restarting": true})
	})
}

// checkForUpdate answers whether a newer release exists, from the remembered
// answer when it is recent enough.
func checkForUpdate(ctx context.Context, current string, force bool) updateCheck {
	result := updateCheck{
		Current:     current,
		Development: update.IsDevelopmentBuild(current),
	}

	stored := loadSettings()
	if !force && stored.UpdateLatest != "" && checkedRecently(stored.UpdateCheckedAt) {
		result.Latest = &releasePayload{Version: stored.UpdateLatest}
		result.Available = updateAvailable(current, stored.UpdateLatest)
		return result
	}

	release, err := fetchLatest(ctx)
	if err != nil {
		// Reported inside the payload rather than as an HTTP failure: this runs
		// unasked on every startup, and being offline is not an error to throw at
		// someone who only wanted to draw their diagrams.
		result.CheckFailed = err.Error()
		return result
	}

	_ = updateSettings(func(s *settings) {
		s.UpdateCheckedAt = time.Now().UTC().Format(time.RFC3339)
		s.UpdateLatest = release.Version
	})

	result.Latest = &releasePayload{
		Version: release.Version,
		Notes:   release.Notes,
		URL:     release.URL,
	}
	if !release.PublishedAt.IsZero() {
		result.Latest.PublishedAt = release.PublishedAt.Format(time.RFC3339)
	}
	result.Available = updateAvailable(current, release.Version)

	// The release page is offered as a link, and no URL crosses from the page:
	// the frontend names "release" and Go resolves it from here.
	rememberReleaseLink(release.URL)

	return result
}

// updateAvailable reports whether latest is worth installing over current.
//
// A development build is left alone: "dev" has no place in the ordering, and
// installing a release over it would replace a binary the user built on purpose.
func updateAvailable(current, latest string) bool {
	if latest == "" || update.IsDevelopmentBuild(current) {
		return false
	}
	return update.Compare(current, latest) < 0
}

func checkedRecently(stamp string) bool {
	at, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return false
	}
	return time.Since(at) < checkInterval
}

// runUpdate downloads and installs, with its progress captured the same way an
// export's is.
func runUpdate(ctx context.Context, cancel context.CancelFunc, run *exportRun, release *update.Release) {
	defer cancel()

	writer := newLineWriter(run.appendLine)
	common.SetOutput(writer)
	defer common.SetOutput(nil)

	installedAt, err := update.Apply(ctx, release)
	writer.Flush()

	failure := ""
	if err != nil && ctx.Err() == nil {
		failure = err.Error()
	}

	if err == nil {
		installedExePath.Lock()
		installedExePath.path = installedAt
		installedExePath.Unlock()
	}

	run.finish(nil, failure)
}
