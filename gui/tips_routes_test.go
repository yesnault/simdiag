package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"simdiag/common"
)

// The cd is the line the whole script depends on. Configurations name their
// templates and output relatively to themselves and the CLI never chdirs, so a
// script without it runs to completion and writes its diagrams beside itself
// instead of into the configured output directory, a failure that reports
// success, which is the worst kind.
func TestBatchScriptContent_MovesToTheConfigurationDirectory(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), configFileName)

	content, err := batchScriptContent(configPath)
	if err != nil {
		t.Fatalf("batchScriptContent: %v", err)
	}

	wantCd := "cd /d " + quoted(filepath.Dir(configPath))
	if !strings.Contains(content, wantCd) {
		t.Errorf("script does not move to the configuration directory\nwant line: %s\ngot:\n%s", wantCd, content)
	}
	if !strings.Contains(content, quoted(configPath)) {
		t.Errorf("script does not name the configuration file:\n%s", content)
	}
}

func TestBatchScriptContent_RunsTheExecutableByAbsolutePath(t *testing.T) {
	content, err := batchScriptContent(filepath.Join(t.TempDir(), configFileName))
	if err != nil {
		t.Fatalf("batchScriptContent: %v", err)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Skip("no executable path on this platform")
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	// After the cd, a bare simdiag.exe is no longer resolvable: the working
	// directory has moved and the binary is not on PATH.
	if !strings.Contains(content, quoted(exe)) {
		t.Errorf("script does not name the executable absolutely:\n%s", content)
	}
	if !strings.Contains(content, " -b ") {
		t.Errorf("script does not run a batch export:\n%s", content)
	}
}

// Without pause the console window closes on the last line and the user never
// sees whether the export worked.
func TestBatchScriptContent_EndsOnPause(t *testing.T) {
	content, err := batchScriptContent(filepath.Join(t.TempDir(), configFileName))
	if err != nil {
		t.Fatalf("batchScriptContent: %v", err)
	}

	if !strings.HasSuffix(strings.TrimRight(content, "\r\n"), "pause") {
		t.Errorf("script does not end on pause:\n%s", content)
	}
	if !strings.Contains(content, batchScriptMarker) {
		t.Errorf("script carries no SimDiag marker, so it could never be recognised again:\n%s", content)
	}
	if !strings.Contains(content, "\r\n") {
		t.Errorf("script does not use CRLF, which cmd.exe expects:\n%q", content)
	}
}

func TestBatchScriptDirectory_FallsBackToTheConfiguration(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, configFileName)

	// The executable directory is the test binary's, which is writable. What
	// matters is that neither answer is empty and that the fallback is the
	// configuration's own directory.
	if got := batchScriptDirectory(configPath); got == "" {
		t.Fatal("batchScriptDirectory returned nothing")
	}

	if exeDir() == "" || !dirWritable(exeDir()) {
		if got := batchScriptDirectory(configPath); got != configDir {
			t.Errorf("batchScriptDirectory = %q, want the configuration directory %q", got, configDir)
		}
	}
}

func TestBatchFileRoute_RefusesToReplaceAForeignScript(t *testing.T) {
	state, script := stateWithBatchScriptIn(t)

	if err := os.WriteFile(script, []byte("@echo off\nmy own flags\n"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got := postBatchFile(t, state, `{}`)
	if got.Created || !got.Exists {
		t.Errorf("response = %+v, want a refusal reporting the existing file", got)
	}

	data, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if !strings.Contains(string(data), "my own flags") {
		t.Errorf("the user's script was overwritten anyway: %q", data)
	}
}

func TestBatchFileRoute_ReplacesAForeignScriptWhenTold(t *testing.T) {
	state, script := stateWithBatchScriptIn(t)

	if err := os.WriteFile(script, []byte("@echo off\nmy own flags\n"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got := postBatchFile(t, state, `{"overwrite":true}`)
	if !got.Created {
		t.Errorf("response = %+v, want the file to have been created", got)
	}

	data, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if !strings.Contains(string(data), batchScriptMarker) {
		t.Errorf("script was not replaced by a generated one: %q", data)
	}
}

// A script SimDiag wrote is ours to refresh, so no question is asked.
func TestBatchFileRoute_RewritesItsOwnScriptWithoutAsking(t *testing.T) {
	state, script := stateWithBatchScriptIn(t)

	if err := os.WriteFile(script, []byte("@echo off\r\n"+batchScriptMarker+"\r\nstale\r\n"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got := postBatchFile(t, state, `{}`)
	if !got.Created || got.Exists {
		t.Errorf("response = %+v, want it rewritten without a question", got)
	}

	data, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if strings.Contains(string(data), "stale") {
		t.Errorf("the stale script was kept: %q", data)
	}
}

// stateWithBatchScriptIn builds a State whose configuration directory is where
// the script will be written, keeping the test off the real install. The
// executable running the tests is go test's binary in a temporary directory,
// which is writable, and the nominal branch would otherwise land there.
func stateWithBatchScriptIn(t *testing.T) (*State, string) {
	t.Helper()

	state := newTestState(t, &common.Config{})
	script := filepath.Join(batchScriptDirectory(state.ConfigPath()), batchScriptName)
	t.Cleanup(func() { os.Remove(script) })

	// Nothing should open a file manager window on the machine running the tests.
	previous := revealFile
	revealFile = func(string) error { return nil }
	t.Cleanup(func() { revealFile = previous })

	return state, script
}

func postBatchFile(t *testing.T, state *State, body string) batchScriptResponse {
	t.Helper()

	rec := httptest.NewRecorder()
	NewHandler(state).ServeHTTP(rec,
		httptest.NewRequest(http.MethodPost, "/api/tips/batch-file", strings.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/tips/batch-file = %d: %s", rec.Code, rec.Body.String())
	}

	var got batchScriptResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return got
}

// The table of links is the allow list: nothing the frontend says can widen it.
func TestOpenURLRoute_RefusesAnythingOutsideTheTable(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{}))

	for _, body := range []string{
		`{"target":"https://example.com/"}`,
		`{"target":"file:///C:/Windows/System32/calc.exe"}`,
		`{"target":"unknown"}`,
		`{"target":""}`,
		`{}`,
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/open-url", strings.NewReader(body)))

		if rec.Code != http.StatusBadRequest {
			t.Errorf("POST /api/open-url %s = %d, want 400", body, rec.Code)
		}
	}
}

// Every destination the frontend names must be in the table, and every URL in
// the table must be one. A typo here sends the user's browser somewhere else.
func TestExternalLinks_AreTheOnesTheInterfaceOffers(t *testing.T) {
	html := readFrontend(t, "index.html")
	js := readFrontend(t, "app.js")

	for _, target := range []string{"drawio", "openkneeboard"} {
		if _, ok := externalLinks[target]; !ok {
			t.Errorf("the interface offers %q, which externalLinks does not have", target)
		}
		if !strings.Contains(html, `data-link="`+target+`"`) && !strings.Contains(js, `target: "`+target+`"`) {
			t.Errorf("externalLinks has %q, which nothing in the interface offers", target)
		}
	}

	for target, url := range externalLinks {
		if !strings.HasPrefix(url, "https://") {
			t.Errorf("%s = %q, want an https URL", target, url)
		}
	}
}
