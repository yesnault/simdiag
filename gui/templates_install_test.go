package gui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"simdiag/common"
	"simdiag/templates"
)

// postInstall asks the route to write the base templates.
func postInstall(t *testing.T, h http.Handler) (installedPayload, int) {
	t.Helper()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/templates/install", nil))

	var payload installedPayload
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decoding the install response: %v", err)
		}
	}
	return payload, rec.Code
}

// The status has to say what is missing before the tab can offer anything.
func TestBaseTemplatesStatus_ReportsWhatIsMissing(t *testing.T) {
	isolateGUIEnvironment(t)

	empty := t.TempDir()
	status := computeStatus(&common.Config{TemplatesDirectory: empty})

	if len(status.BaseTemplates.Missing) != len(templates.Names()) {
		t.Errorf("missing = %v, want all %d", status.BaseTemplates.Missing, len(templates.Names()))
	}
	if status.BaseTemplates.Target != empty {
		t.Errorf("target = %q, want the configured directory %q", status.BaseTemplates.Target, empty)
	}

	if _, err := templates.Install(empty); err != nil {
		t.Fatalf("Install: %v", err)
	}

	after := computeStatus(&common.Config{TemplatesDirectory: empty})
	if len(after.BaseTemplates.Missing) != 0 {
		t.Errorf("missing = %v after installing, want none", after.BaseTemplates.Missing)
	}
	// Never null: the frontend reads the length before anything else.
	if after.BaseTemplates.Missing == nil {
		t.Error("missing is null, want an empty list")
	}
}

// With nothing configured, the templates go beside the configuration file: the
// same "templates" the form already suggests.
func TestBaseTemplatesStatus_FallsBackToTheDefaultDirectory(t *testing.T) {
	isolateGUIEnvironment(t)

	status := computeStatus(&common.Config{})

	if status.BaseTemplates.Target != defaultTemplatesDirectory {
		t.Errorf("target = %q, want %q", status.BaseTemplates.Target, defaultTemplatesDirectory)
	}
}

func TestInstallRoute_WritesTheTemplatesAndRecordsTheDirectory(t *testing.T) {
	isolateGUIEnvironment(t) // also moves the working directory to a temp dir

	state := newTestState(t, &common.Config{})
	h := NewHandler(state)

	payload, code := postInstall(t, h)
	if code != http.StatusOK {
		t.Fatalf("POST /api/templates/install = %d, want 200", code)
	}

	if payload.Installed != len(templates.Names()) {
		t.Errorf("installed %d, want %d", payload.Installed, len(templates.Names()))
	}
	for _, name := range templates.Names() {
		if _, err := os.Stat(filepath.Join(payload.Target, name)); err != nil {
			t.Errorf("%q was not written: %v", name, err)
		}
	}

	// Files without the setting would leave the user just as stuck.
	if got := state.Config().TemplatesDirectory; got != defaultTemplatesDirectory {
		t.Errorf("TemplatesDirectory = %q, want it recorded as %q", got, defaultTemplatesDirectory)
	}
	if payload.Config.Status.Templates.Severity != "" {
		t.Errorf("templates status = %+v, want it clean after installing", payload.Config.Status.Templates)
	}
}

// A configured directory is used as it is, and an existing template is left
// alone. Installing is never allowed to take back someone's edit.
func TestInstallRoute_KeepsAnEditedTemplate(t *testing.T) {
	isolateGUIEnvironment(t)

	dir := t.TempDir()
	edited := templates.Names()[0]
	const mine = "<svg>mine</svg>"
	if err := os.WriteFile(filepath.Join(dir, edited), []byte(mine), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	h := NewHandler(newTestState(t, &common.Config{TemplatesDirectory: dir}))

	payload, code := postInstall(t, h)
	if code != http.StatusOK {
		t.Fatalf("POST /api/templates/install = %d, want 200", code)
	}
	if payload.Installed != len(templates.Names())-1 {
		t.Errorf("installed %d, want all but the edited one", payload.Installed)
	}

	after, err := os.ReadFile(filepath.Join(dir, edited))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(after) != mine {
		t.Errorf("%q was overwritten", edited)
	}
}

// The install writes under a root a running export is reading from.
func TestInstallRoute_RefusedWhileExporting(t *testing.T) {
	isolateGUIEnvironment(t)

	h := NewHandler(newTestState(t, &common.Config{TemplatesDirectory: t.TempDir()}))

	_, cancel := context.WithCancel(context.Background())
	if !currentExport.begin(cancel) {
		t.Fatal("setup: could not claim the export slot")
	}
	t.Cleanup(func() { currentExport.finish(nil, "") })

	if _, code := postInstall(t, h); code != http.StatusConflict {
		t.Errorf("install during an export = %d, want 409", code)
	}
}
