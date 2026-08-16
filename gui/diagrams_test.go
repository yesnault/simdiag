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

// buildOutputTree lays out what an export actually leaves behind: one directory
// per simulator or DCS module, SVG files inside, PNG copies under png/.
func buildOutputTree(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{"dcs-m2000c", "il2", "il2-korea"} {
		if err := os.MkdirAll(filepath.Join(root, dir, "png"), 0755); err != nil {
			t.Fatalf("setup: %v", err)
		}
		for _, name := range []string{"Joystick.svg", "Throttle.svg"} {
			if err := os.WriteFile(filepath.Join(root, dir, name), []byte("<svg/>"), 0644); err != nil {
				t.Fatalf("setup: %v", err)
			}
		}
		// Only one of the two gets a PNG, covering the "no PNG" case.
		if err := os.WriteFile(filepath.Join(root, dir, "png", "Joystick.png"), []byte("png"), 0644); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "export.csv"), []byte("Simulator\n"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	return root
}

func TestScanDiagrams_GroupsBySubdirectory(t *testing.T) {
	payload := scanDiagrams(&common.Config{OutputDirectory: buildOutputTree(t)})

	if len(payload.Groups) != 3 {
		t.Fatalf("groups = %d, want 3: %+v", len(payload.Groups), payload.Groups)
	}
	if !payload.HasCSV {
		t.Error("hasCsv = false, want true")
	}

	var labels []string
	for _, group := range payload.Groups {
		labels = append(labels, group.Label)
		if len(group.Diagrams) != 2 {
			t.Errorf("group %q has %d diagram(s), want 2", group.Label, len(group.Diagrams))
		}
	}

	joined := strings.Join(labels, "|")
	for _, want := range []string{"DCS World · m2000c", "IL-2 Sturmovik", "IL-2 Korea"} {
		if !strings.Contains(joined, want) {
			t.Errorf("labels = %q, want one reading %q", joined, want)
		}
	}
}

func TestScanDiagrams_LinksPNGWhenPresent(t *testing.T) {
	payload := scanDiagrams(&common.Config{OutputDirectory: buildOutputTree(t)})

	var withPNG, withoutPNG int
	for _, group := range payload.Groups {
		for _, diagram := range group.Diagrams {
			if diagram.PNGPath != "" {
				withPNG++
			} else {
				withoutPNG++
			}
		}
	}

	if withPNG != 3 || withoutPNG != 3 {
		t.Errorf("PNG links: %d with, %d without; want 3 and 3", withPNG, withoutPNG)
	}
}

// Paths must use forward slashes: they end up in a URL query parameter.
func TestScanDiagrams_UsesSlashPaths(t *testing.T) {
	payload := scanDiagrams(&common.Config{OutputDirectory: buildOutputTree(t)})

	for _, group := range payload.Groups {
		for _, diagram := range group.Diagrams {
			if strings.Contains(diagram.Path, "\\") {
				t.Errorf("path %q contains a backslash", diagram.Path)
			}
			if diagram.PNGPath != "" && strings.Contains(diagram.PNGPath, "\\") {
				t.Errorf("png path %q contains a backslash", diagram.PNGPath)
			}
		}
	}
}

func TestScanDiagrams_MissingOutputDirectory(t *testing.T) {
	payload := scanDiagrams(&common.Config{
		OutputDirectory: filepath.Join(t.TempDir(), "never-exported"),
	})

	if len(payload.Groups) != 0 {
		t.Errorf("groups = %d, want 0", len(payload.Groups))
	}
	if len(payload.Warnings) == 0 {
		t.Error("want a warning telling the user to run an export")
	}
}

func TestScanDiagrams_UnconfiguredOutputDirectory(t *testing.T) {
	payload := scanDiagrams(&common.Config{})

	if len(payload.Warnings) == 0 {
		t.Error("want a warning when the output directory is not configured")
	}
}

// Every group but one is named after a simulator or a module: proper nouns,
// shown as they are. The output directory itself is a phrase, so it travels as
// a code and the interface translates it.
func TestDiagramGroupLabel(t *testing.T) {
	cases := map[string]string{
		"":           "",
		"il2":        "IL-2 Sturmovik",
		"il2-korea":  "IL-2 Korea",
		"dcs-fa18c":  "DCS World · fa18c",
		"unexpected": "unexpected",
	}

	for subdir, want := range cases {
		if got := diagramGroupLabel(subdir); got != want {
			t.Errorf("diagramGroupLabel(%q) = %q, want %q", subdir, got, want)
		}
	}

	if got := diagramGroupLabelCode(""); got != msgOutputGroup {
		t.Errorf("diagramGroupLabelCode(\"\") = %q, want %q", got, msgOutputGroup)
	}
	if got := diagramGroupLabelCode("il2"); got != "" {
		t.Errorf("diagramGroupLabelCode(%q) = %q, want no code: it is a proper noun", "il2", got)
	}
}

func TestDiagramRoute_ServesTheGallery(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{OutputDirectory: buildOutputTree(t)}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/diagrams", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/diagrams = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Joystick.svg") {
		t.Errorf("gallery body = %s", rec.Body.String())
	}
}

// Regenerating needs a CSV to work from; asking without one is a clear refusal
// rather than an empty run.
func TestDiagramRoute_RegenerateRequiresACSV(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{
		OutputDirectory:    t.TempDir(),
		TemplatesDirectory: t.TempDir(),
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diagrams/regenerate", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("regenerate without a CSV = %d, want 400", rec.Code)
	}
}

// "Open folder" runs a shell command on a path from the frontend. It must
// refuse anything outside the output directory.
func TestDiagramRoute_OpenRefusesPathsOutsideOutput(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{OutputDirectory: buildOutputTree(t)}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diagrams/open",
		strings.NewReader(`{"path":"../../../Windows/System32"}`)))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("open outside the output directory = %d, want 400", rec.Code)
	}
}

// The toolbar button opens the output directory itself, which it asks for as
// ".": it sits next to that path, and opening the first module's folder instead
// was what it used to do.
func TestOpenPath_DotIsTheOutputDirectoryItself(t *testing.T) {
	output := buildOutputTree(t)

	target, err := safeJoin(output, ".")
	if err != nil {
		t.Fatalf("safeJoin(output, %q): %v", ".", err)
	}

	want, err := filepath.Abs(output)
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	if target != want {
		t.Errorf("safeJoin(output, \".\") = %q, want the output directory %q", target, want)
	}
}

// Explorer swallows its own failures on Windows. A path that is not there
// has to be reported rather than looking like a click that did nothing.
func TestDiagramRoute_OpenReportsAMissingFolder(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{OutputDirectory: t.TempDir()}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diagrams/open",
		strings.NewReader(`{"path":"dcs-nope/absent.svg"}`)))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("open of a missing path = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), msgFolderMissing) {
		t.Errorf("body = %q, want the %s code", rec.Body.String(), msgFolderMissing)
	}
}

func TestDiagramRoute_OpenRejectsInvalidPayload(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{OutputDirectory: t.TempDir()}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/diagrams/open", strings.NewReader("nope")))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid payload = %d, want 400", rec.Code)
	}
}
