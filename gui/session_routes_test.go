package gui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"simdiag/common"
)

// postSession sends a request to a picker route and decodes the session.
func postSession(t *testing.T, h http.Handler, route, body string) (sessionPayload, int) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, route, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var payload sessionPayload
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decoding %s: %v", route, err)
		}
	}
	return payload, rec.Code
}

func TestSession_ReportsAListEvenWhenEmpty(t *testing.T) {
	isolateGUIEnvironment(t)

	h := NewHandler(newTestState(t, &common.Config{}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/session", nil))

	// The frontend reads recent.length before anything else, and null is not
	// iterable on the other side.
	if !strings.Contains(rec.Body.String(), `"recent":[`) {
		t.Errorf("GET /api/session body = %q, want recent as a list", rec.Body.String())
	}
}

func TestLanguage_IsStoredAndReported(t *testing.T) {
	isolateGUIEnvironment(t)

	h := NewHandler(newTestState(t, &common.Config{}))

	payload, code := postSession(t, h, "/api/language", `{"language":"fr"}`)
	if code != http.StatusOK {
		t.Fatalf("POST /api/language = %d, want 200", code)
	}
	if payload.Language != "fr" {
		t.Errorf("language = %q, want fr", payload.Language)
	}

	// And it is still there on the next session, which is the whole point.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/session", nil))
	if !strings.Contains(rec.Body.String(), `"language":"fr"`) {
		t.Errorf("GET /api/session body = %q, want the stored language", rec.Body.String())
	}
}

func TestLanguage_RejectsOneWeDoNotHave(t *testing.T) {
	isolateGUIEnvironment(t)

	h := NewHandler(newTestState(t, &common.Config{}))

	if _, code := postSession(t, h, "/api/language", `{"language":"de"}`); code != http.StatusBadRequest {
		t.Errorf("POST /api/language with an untranslated language = %d, want 400", code)
	}
	if got := loadSettings().Language; got != "" {
		t.Errorf("language = %q, want nothing stored", got)
	}
}

// A refusal the user is meant to act on comes back as a message code, so the
// interface can show it in their language rather than in English.
func TestSwitchingWhileExporting_ReportsAMessageCode(t *testing.T) {
	isolateGUIEnvironment(t)

	h := NewHandler(newTestState(t, &common.Config{}))

	_, cancel := context.WithCancel(context.Background())
	if !currentExport.begin(cancel) {
		t.Fatal("setup: could not claim the export slot")
	}
	t.Cleanup(func() { currentExport.finish(nil, "") })

	req := httptest.NewRequest(http.MethodPost, "/api/config/reload", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("POST /api/config/reload = %d, want 409", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), msgExportRunning) {
		t.Errorf("body = %q, want the %s code", rec.Body.String(), msgExportRunning)
	}
}

func TestOpenConfig_SwitchesAndRemembers(t *testing.T) {
	isolateGUIEnvironment(t)

	state := newTestState(t, &common.Config{})
	h := NewHandler(state)

	other := writeConfigFile(t, t.TempDir(), configFileName)

	payload, code := postSession(t, h, "/api/config/open", `{"path":`+quote(other)+`}`)
	if code != http.StatusOK {
		t.Fatalf("POST /api/config/open = %d, want 200", code)
	}
	if payload.ConfigPath != other {
		t.Errorf("configPath = %q, want %q", payload.ConfigPath, other)
	}
	if len(payload.Recent) == 0 || payload.Recent[0].Path != other {
		t.Errorf("recent = %+v, want the file just opened at the top", payload.Recent)
	}
}

// Requests coming through the Wails asset server do not always declare a
// Content-Length. A handler that skips decoding on that basis would drop the
// path and silently open the file dialog instead of the file that was asked for.
func TestOpenConfig_ReadsABodyWithoutAContentLength(t *testing.T) {
	isolateGUIEnvironment(t)

	state := newTestState(t, &common.Config{})
	h := NewHandler(state)

	other := writeConfigFile(t, t.TempDir(), configFileName)

	req := httptest.NewRequest(http.MethodPost, "/api/config/open", strings.NewReader(`{"path":`+quote(other)+`}`))
	req.ContentLength = -1
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/config/open = %d, want 200", rec.Code)
	}
	if state.ConfigPath() != other {
		t.Errorf("ConfigPath() = %q, want %q", state.ConfigPath(), other)
	}
}

func TestOpenConfig_RejectsAFileThatIsNotThere(t *testing.T) {
	isolateGUIEnvironment(t)

	state := newTestState(t, &common.Config{})
	h := NewHandler(state)
	before := state.ConfigPath()

	absent := filepath.Join(t.TempDir(), "absent.yaml")
	if _, code := postSession(t, h, "/api/config/open", `{"path":`+quote(absent)+`}`); code != http.StatusBadRequest {
		t.Errorf("POST /api/config/open on a missing file = %d, want 400", code)
	}
	if state.ConfigPath() != before {
		t.Errorf("ConfigPath() = %q after a refused switch, want %q", state.ConfigPath(), before)
	}
}

// An export holds a snapshot of the old configuration and reads it for a minute
// or more; moving the working directory out from under it would break it.
func TestSwitchingIsRefusedWhileAnExportRuns(t *testing.T) {
	isolateGUIEnvironment(t)

	h := NewHandler(newTestState(t, &common.Config{}))

	_, cancel := context.WithCancel(context.Background())
	if !currentExport.begin(cancel) {
		t.Fatal("setup: could not claim the export slot")
	}
	t.Cleanup(func() { currentExport.finish(nil, "") })

	other := writeConfigFile(t, t.TempDir(), configFileName)
	if _, code := postSession(t, h, "/api/config/open", `{"path":`+quote(other)+`}`); code != http.StatusConflict {
		t.Errorf("POST /api/config/open during an export = %d, want 409", code)
	}
	if _, code := postSession(t, h, "/api/config/reload", ""); code != http.StatusConflict {
		t.Errorf("POST /api/config/reload during an export = %d, want 409", code)
	}
}

func TestReloadConfig_ReadsTheFileAgain(t *testing.T) {
	isolateGUIEnvironment(t)

	state := newTestState(t, &common.Config{OutputDirectory: "./before"})
	h := NewHandler(state)

	// Someone edited the file in a text editor while the application was open.
	if err := common.SaveConfigTo(&common.Config{OutputDirectory: "./after"}, state.ConfigPath()); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if _, code := postSession(t, h, "/api/config/reload", ""); code != http.StatusOK {
		t.Fatalf("POST /api/config/reload = %d, want 200", code)
	}
	if got := state.Config().OutputDirectory; got != "./after" {
		t.Errorf("OutputDirectory = %q after a reload, want the file's current value", got)
	}
}

func quote(s string) string {
	encoded, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(encoded)
}
