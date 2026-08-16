package gui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"simdiag/common"
)

func TestSafeJoin_AcceptsPathsInsideRoot(t *testing.T) {
	root := t.TempDir()

	for _, rel := range []string{
		"template.svg",
		"sub/template.svg",
		"sub\\template.svg",
		"./template.svg",
		"sub/../template.svg",
	} {
		got, err := safeJoin(root, rel)
		if err != nil {
			t.Errorf("safeJoin(root, %q) unexpected error: %v", rel, err)
			continue
		}
		if !strings.HasPrefix(got, root) {
			t.Errorf("safeJoin(root, %q) = %q, want it under %q", rel, got, root)
		}
	}
}

func TestSafeJoin_RejectsTraversal(t *testing.T) {
	root := filepath.Join(t.TempDir(), "templates")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	for _, rel := range []string{
		"../secret.txt",
		"../../secret.txt",
		"sub/../../secret.txt",
		"..\\secret.txt",
		"..",
	} {
		if got, err := safeJoin(root, rel); err == nil {
			t.Errorf("safeJoin(root, %q) = %q, want an error", rel, got)
		}
	}
}

func TestSafeJoin_RejectsUnconfiguredRoot(t *testing.T) {
	if _, err := safeJoin("", "template.svg"); err == nil {
		t.Error("safeJoin with an empty root should return an error")
	}
}

// newTestState builds a State backed by a temporary config, without touching
// the user's real configuration file.
func newTestState(t *testing.T, config *common.Config) *State {
	t.Helper()

	path := filepath.Join(t.TempDir(), configFileName)
	if err := common.SaveConfigTo(config, path); err != nil {
		t.Fatalf("setup: %v", err)
	}

	state, err := NewState(path, "test-version")
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	return state
}

func TestHandler_ServesVersion(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/session", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/session = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "test-version") {
		t.Errorf("GET /api/session body = %q, want it to contain the version", rec.Body.String())
	}
}

func TestHandler_ServesEmbeddedFrontend(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "SimDiag") {
		t.Errorf("GET / did not serve the embedded index.html")
	}
}

// The page is an ES module, and a module served with the wrong Content-Type is
// refused by the browser rather than tolerated. net/http guesses the type from
// the Windows registry, so the guess is a property of the machine: without this
// header, SimDiag would open on a blank window on some installs and not others.
func TestHandler_ServesScriptsAsJavaScript(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{}))

	for _, script := range []string{"/app.js", "/i18n.js"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, script, nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", script, rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/javascript") {
			t.Errorf("GET %s Content-Type = %q, want text/javascript", script, got)
		}
	}
}

func TestHandler_ServesTemplateFromConfiguredDirectory(t *testing.T) {
	templates := t.TempDir()
	if err := os.WriteFile(filepath.Join(templates, "device.svg"), []byte("<svg>ok</svg>"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	h := NewHandler(newTestState(t, &common.Config{TemplatesDirectory: templates}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/template?p=device.svg", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/template = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "<svg>ok</svg>") {
		t.Errorf("GET /api/template body = %q", body)
	}
}

// The webview is a browser: a template path that escapes the configured
// directory must never be served.
func TestHandler_RefusesTemplateOutsideRoot(t *testing.T) {
	base := t.TempDir()
	templates := filepath.Join(base, "templates")
	if err := os.MkdirAll(templates, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "secret.txt"), []byte("secret"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	h := NewHandler(newTestState(t, &common.Config{TemplatesDirectory: templates}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/template?p=../secret.txt", nil))

	if rec.Code == http.StatusOK {
		t.Fatalf("GET /api/template?p=../secret.txt = 200, want a refusal")
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("traversal leaked file content: %q", rec.Body.String())
	}
}

func TestHandler_RequiresPathParameter(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{TemplatesDirectory: t.TempDir()}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/template", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET /api/template without p = %d, want 400", rec.Code)
	}
}

// writeFile creates an empty file for status tests.
func writeFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("<svg/>"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
}
