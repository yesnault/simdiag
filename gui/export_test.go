package gui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"simdiag/common"
	"simdiag/workflow"
)

func capturedLines() (*lineWriter, *[]string) {
	var got []string
	writer := newLineWriter(func(line string) { got = append(got, line) })
	return writer, &got
}

func TestLineWriter_SplitsOnNewlines(t *testing.T) {
	writer, got := capturedLines()

	if _, err := writer.Write([]byte("first\nsecond\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	if len(*got) != 2 || (*got)[0] != "first" || (*got)[1] != "second" {
		t.Errorf("lines = %v, want [first second]", *got)
	}
}

// The pipeline writes in arbitrary chunks; a line split across two writes must
// still arrive as one line.
func TestLineWriter_JoinsPartialWrites(t *testing.T) {
	writer, got := capturedLines()

	_, _ = writer.Write([]byte("Process"))
	_, _ = writer.Write([]byte("ing DCS"))
	_, _ = writer.Write([]byte(" World\n"))

	if len(*got) != 1 || (*got)[0] != "Processing DCS World" {
		t.Errorf("lines = %v, want [Processing DCS World]", *got)
	}
}

func TestLineWriter_FlushEmitsTrailingText(t *testing.T) {
	writer, got := capturedLines()

	_, _ = writer.Write([]byte("no trailing newline"))
	if len(*got) != 0 {
		t.Fatalf("lines = %v, want nothing before Flush", *got)
	}

	writer.Flush()

	if len(*got) != 1 || (*got)[0] != "no trailing newline" {
		t.Errorf("lines = %v after Flush", *got)
	}
}

func TestLineWriter_StripsCarriageReturns(t *testing.T) {
	writer, got := capturedLines()

	_, _ = writer.Write([]byte("windows line\r\n"))

	if len(*got) != 1 || (*got)[0] != "windows line" {
		t.Errorf("lines = %v, want the \\r stripped", *got)
	}
}

// --- run bookkeeping

func TestExportRun_RefusesASecondRun(t *testing.T) {
	run := &exportRun{}

	if !run.begin(func() {}) {
		t.Fatal("the first run was refused")
	}
	if run.begin(func() {}) {
		t.Error("a second run started while one was in flight")
	}

	run.finish(nil, "")

	if !run.begin(func() {}) {
		t.Error("a run was refused after the previous one finished")
	}
}

// The frontend polls with the index it already has; it must receive only what
// is new, and an index it can reuse.
func TestExportRun_StateSinceReturnsOnlyNewLines(t *testing.T) {
	run := &exportRun{}
	run.begin(func() {})
	run.appendLine("one")
	run.appendLine("two")

	first := run.stateSince(0)
	if len(first.Lines) != 2 || first.NextIndex != 2 {
		t.Fatalf("first poll = %+v, want 2 lines and nextIndex 2", first)
	}

	run.appendLine("three")

	second := run.stateSince(first.NextIndex)
	if len(second.Lines) != 1 || second.Lines[0] != "three" {
		t.Errorf("second poll = %+v, want only the new line", second)
	}
	if !second.Running {
		t.Error("running = false while the run is in flight")
	}
}

// A poll that finds nothing new is the common case, and it must serialise as an
// empty list. JSON null is not iterable in the frontend.
func TestExportRun_StateSinceEncodesAnEmptyListNotNull(t *testing.T) {
	run := &exportRun{}
	run.begin(func() {})
	run.appendLine("one")

	encoded, err := json.Marshal(run.stateSince(1))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"lines":[]`) {
		t.Errorf("payload = %s, want an empty lines array", encoded)
	}
}

// A stale index (from a previous run) must not panic or slice out of range.
func TestExportRun_StateSinceToleratesOutOfRangeIndex(t *testing.T) {
	run := &exportRun{}
	run.appendLine("only one")

	for _, from := range []int{-5, 99} {
		state := run.stateSince(from)
		if len(state.Lines) != 1 {
			t.Errorf("stateSince(%d) = %v, want the whole log", from, state.Lines)
		}
	}
}

func TestExportRun_CancelMarksTheRun(t *testing.T) {
	run := &exportRun{}

	if run.requestCancel() {
		t.Error("cancelling with no run in flight reported success")
	}

	ctx, cancel := context.WithCancel(context.Background())
	run.begin(cancel)

	if !run.requestCancel() {
		t.Fatal("cancelling a running export reported failure")
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("the run's context was not cancelled")
	}
	if !run.stateSince(0).Cancelled {
		t.Error("cancelled = false after a cancel request")
	}
}

// --- routes

func TestBuildExportTargets_AlwaysOffersEverything(t *testing.T) {
	payload := buildExportTargets(&common.Config{})

	if len(payload.Targets) == 0 || payload.Targets[0].Kind != "all" {
		t.Fatalf("targets = %+v, want 'all' first", payload.Targets)
	}
	if len(payload.Warnings) == 0 {
		t.Error("want a warning when templates and output directories are unset")
	}
}

// makeDCSInstall builds the Config/Input layout the module detector reads, and
// returns the path to use as dcs_path.
func makeDCSInstall(t *testing.T, profiles ...string) string {
	t.Helper()

	base := t.TempDir()
	for _, name := range profiles {
		if err := os.MkdirAll(filepath.Join(base, "Config", "Input", name, "joystick"), 0755); err != nil {
			t.Fatalf("setup %s: %v", name, err)
		}
	}
	return base
}

// The dropdown's DCS entries come from the installation on disk, not from the
// configuration, which names no module at all.
func TestBuildExportTargets_ListsModulesAndSimulators(t *testing.T) {
	config := &common.Config{
		TemplatesDirectory: "templates",
		OutputDirectory:    "output",
		Simulators: map[string]*common.SimulatorConfig{
			"dcs_world":     {DCSPath: makeDCSInstall(t, "M-2000C", "FA-18C_hornet", "Default")},
			"il2_sturmovik": {IL2InputPath: `C:\il2`},
		},
	}

	payload := buildExportTargets(config)

	var labels []string
	for _, target := range payload.Targets {
		labels = append(labels, target.Label)
	}
	joined := strings.Join(labels, "|")

	for _, want := range []string{"M-2000C", "FA-18C_hornet", "IL-2 Sturmovik"} {
		if !strings.Contains(joined, want) {
			t.Errorf("targets = %q, want an entry for %q", joined, want)
		}
	}
	// Default is a shared profile, not an aircraft.
	if strings.Contains(joined, "Default") {
		t.Errorf("targets = %q, Default is not a module", joined)
	}
	// DCS is represented by its modules, not as a whole-simulator entry.
	if strings.Contains(joined, "|DCS World|") {
		t.Errorf("targets = %q, DCS should appear only per module", joined)
	}
	if len(payload.Warnings) != 0 {
		t.Errorf("warnings = %v, want none for a complete config", payload.Warnings)
	}
}

// A dropdown must never offer a target the export would skip. Detection resolves
// the DCS path the same way the export does, and that resolution falls back to
// the stock Saved Games location, so on a machine with DCS installed but not
// configured, an ungated detector would list its modules here.
func TestBuildExportTargets_NoModulesWithoutDCSPath(t *testing.T) {
	for _, tc := range []struct {
		name string
		dcs  *common.SimulatorConfig
	}{
		{"no DCS section at all", nil},
		{"a section without a path", &common.SimulatorConfig{GremlinsProfileFilepath: `C:\gremlins.xml`}},
		{"a path that does not resolve", &common.SimulatorConfig{DCSPath: filepath.Join(t.TempDir(), "nope")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := &common.Config{
				TemplatesDirectory: "templates",
				OutputDirectory:    "output",
				Simulators:         map[string]*common.SimulatorConfig{},
			}
			if tc.dcs != nil {
				config.Simulators["dcs_world"] = tc.dcs
			}

			for _, target := range buildExportTargets(config).Targets {
				if target.Kind == "module" {
					t.Errorf("unexpected module target %q", target.Filter)
				}
			}
		})
	}
}

// The dropdown sends the raw profile folder name, and the workflow filters
// against that same name. Each target must select its own module and no other,
// or Generate runs to completion and exports nothing.
func TestBuildExportTargets_ModuleFiltersSelectTheirModule(t *testing.T) {
	profiles := []string{"M-2000C", "FA-18C_hornet", "P-47D-30"}

	payload := buildExportTargets(&common.Config{
		TemplatesDirectory: "templates",
		OutputDirectory:    "output",
		Simulators: map[string]*common.SimulatorConfig{
			"dcs_world": {DCSPath: makeDCSInstall(t, profiles...)},
		},
	})

	seen := 0
	for _, target := range payload.Targets {
		if target.Kind != "module" {
			continue
		}
		seen++

		for _, profileName := range profiles {
			want := profileName == target.Filter
			if got := workflow.MatchesModuleFilter(profileName, target.Filter); got != want {
				t.Errorf("target %q against profile %q = %v, want %v",
					target.Filter, profileName, got, want)
			}
		}
	}

	if seen != len(profiles) {
		t.Errorf("module targets = %d, want %d", seen, len(profiles))
	}
}

// A CLI user types the normalized key, so the same target must still match it.
func TestBuildExportTargets_ModuleFiltersAcceptTheNormalizedKey(t *testing.T) {
	payload := buildExportTargets(&common.Config{
		TemplatesDirectory: "templates",
		OutputDirectory:    "output",
		Simulators: map[string]*common.SimulatorConfig{
			"dcs_world": {DCSPath: makeDCSInstall(t, "M-2000C")},
		},
	})

	for _, target := range payload.Targets {
		if target.Kind != "module" {
			continue
		}
		if !workflow.MatchesModuleFilter(target.Filter, "m2000c") {
			t.Errorf("target %q is not selected by the normalized key m2000c", target.Filter)
		}
	}
}

// The dropdown is rebuilt on every load; an unsorted directory read would
// reshuffle it.
func TestBuildExportTargets_ModulesAreSorted(t *testing.T) {
	payload := buildExportTargets(&common.Config{
		TemplatesDirectory: "templates",
		OutputDirectory:    "output",
		Simulators: map[string]*common.SimulatorConfig{
			"dcs_world": {DCSPath: makeDCSInstall(t, "P-47D-30", "M-2000C", "FA-18C_hornet")},
		},
	})

	var filters []string
	for _, target := range payload.Targets {
		if target.Kind == "module" {
			filters = append(filters, target.Filter)
		}
	}

	if !slices.IsSorted(filters) {
		t.Errorf("module targets = %v, want them sorted", filters)
	}
}

func TestExportRoute_RejectsInvalidPayload(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/export/start", strings.NewReader("not json")))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid payload = %d, want 400", rec.Code)
	}
}

// End to end over HTTP: start a run, poll until it finishes, and check the log
// and summary actually arrived.
func TestExportRoute_StartPollAndFinish(t *testing.T) {
	state := newTestState(t, &common.Config{
		TemplatesDirectory: t.TempDir(),
		OutputDirectory:    t.TempDir(),
	})
	h := NewHandler(state)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/export/start", strings.NewReader(`{"noSvg":true}`)))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("start = %d (%s), want 202", rec.Code, rec.Body.String())
	}

	var latest exportState
	from := 0
	deadline := time.After(20 * time.Second)

	for {
		poll := httptest.NewRecorder()
		h.ServeHTTP(poll, httptest.NewRequest(http.MethodGet,
			"/api/export/state?from="+strconv.Itoa(from), nil))

		if poll.Code != http.StatusOK {
			t.Fatalf("poll = %d (%s)", poll.Code, poll.Body.String())
		}
		if err := json.Unmarshal(poll.Body.Bytes(), &latest); err != nil {
			t.Fatalf("poll body is not valid JSON: %v", err)
		}
		from = latest.NextIndex

		if !latest.Running {
			break
		}

		select {
		case <-deadline:
			t.Fatal("the export never finished")
		case <-time.After(20 * time.Millisecond):
		}
	}

	if from == 0 {
		t.Error("the run produced no log line")
	}
	if latest.Summary == nil {
		t.Error("the finished run carried no summary")
	}
	if latest.Error != "" {
		t.Errorf("run reported an error: %s", latest.Error)
	}
}

// The run must hand the progress writer back, or every later CLI-style print in
// the process would keep going to a finished run's buffer.
func TestExportRoute_RestoresProgressOutput(t *testing.T) {
	run := &exportRun{}
	ctx, cancel := context.WithCancel(context.Background())
	run.begin(cancel)

	runExport(ctx, cancel, run, &common.Config{
		TemplatesDirectory: t.TempDir(),
		OutputDirectory:    t.TempDir(),
	}, exportRequest{NoSVG: true})

	var sink strings.Builder
	common.SetOutput(&sink)
	common.Printf("after the run\n")
	common.SetOutput(nil)

	if !strings.Contains(sink.String(), "after the run") {
		t.Error("common output was not restored after the export")
	}
}
