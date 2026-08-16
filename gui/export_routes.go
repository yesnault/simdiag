package gui

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"simdiag/app"
	"simdiag/common"
	"simdiag/workflow"
)

// exportRequest is what the Generate tab sends to start a run.
type exportRequest struct {
	Filter string `json:"filter"`
	NoSVG  bool   `json:"noSvg"`
}

// exportTargetsPayload lists what the user can choose to export.
type exportTargetsPayload struct {
	Targets  []exportTarget `json:"targets"`
	DrawIO   pathStatus     `json:"drawio"`
	Warnings []message      `json:"warnings"`
}

// exportTarget is one entry of the "what to export" selector. The engine takes a
// single substring filter, not a set, so this is a single choice rather than a
// list of checkboxes.
// Label is a proper noun shown as is; LabelCode names a translated label and
// wins when set: only the "everything" entry needs one.
type exportTarget struct {
	Label     string `json:"label"`
	LabelCode string `json:"labelCode,omitempty"`
	Filter    string `json:"filter"`
	Kind      string `json:"kind"` // "all", "module" or "simulator"
}

// Only one export at a time: a run redirects the process-wide progress writer,
// so two concurrent runs would interleave their logs into each other's buffer.
var currentExport = &exportRun{}

func registerExportRoutes(mux *http.ServeMux, state *State) {
	mux.HandleFunc("GET /api/export/targets", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, buildExportTargets(state.Config()))
	})

	mux.HandleFunc("POST /api/export/start", func(w http.ResponseWriter, r *http.Request) {
		var req exportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Work from a copy: the export reads the configuration for seconds,
		// during which the user can still save the Configuration tab.
		config, err := state.ConfigSnapshot()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		if !currentExport.begin(cancel) {
			cancel()
			writeMessageError(w, http.StatusConflict, msgExportRunning, nil)
			return
		}

		go runExport(ctx, cancel, currentExport, config, req)

		w.WriteHeader(http.StatusAccepted)
		writeJSON(w, currentExport.stateSince(0))
	})

	// The Generate tab polls this while a run is in flight.
	mux.HandleFunc("GET /api/export/state", func(w http.ResponseWriter, r *http.Request) {
		from, _ := strconv.Atoi(r.URL.Query().Get("from"))
		writeJSON(w, currentExport.stateSince(from))
	})

	mux.HandleFunc("POST /api/export/cancel", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]bool{"cancelled": currentExport.requestCancel()})
	})
}

// runExport executes one export, recording its progress on the run.
func runExport(ctx context.Context, cancel context.CancelFunc, run *exportRun, config *common.Config, req exportRequest) {
	defer cancel()

	writer := newLineWriter(run.appendLine)

	// Redirect the pipeline's progress output for the duration of the run.
	common.SetOutput(writer)
	defer common.SetOutput(nil)

	summary, err := workflow.ExportAll(ctx, config, app.Parsers(config), app.Enrichers(), req.Filter, req.NoSVG)
	writer.Flush()

	failure := ""
	if err != nil && ctx.Err() == nil {
		failure = err.Error()
	}

	run.finish(summary, failure)
}

// buildExportTargets offers "everything", then each configured DCS module, then
// each configured simulator.
func buildExportTargets(config *common.Config) exportTargetsPayload {
	payload := exportTargetsPayload{
		Targets: []exportTarget{{LabelCode: msgExportTargetEverything, Filter: "", Kind: "all"}},
		DrawIO:  drawIOStatus(config),
	}

	// Detected on disk, not read from the configuration: listing Config/Input is
	// a handful of stat calls, so the dropdown can offer the real modules
	// without paying for a parse every time the tab opens.
	for _, module := range app.DetectDCSModules(config) {
		payload.Targets = append(payload.Targets, exportTarget{
			Label:  "DCS World · " + module,
			Filter: module,
			Kind:   "module",
		})
	}

	for _, simType := range simulatorOrder {
		// DCS is covered by its per-module entries above.
		if simType == common.DCSWorld {
			continue
		}
		section := config.LookupSimulatorConfig(simType)
		if section == nil || section.IL2InputPath == "" {
			continue
		}
		payload.Targets = append(payload.Targets, exportTarget{
			Label:  string(simType),
			Filter: string(simType),
			Kind:   "simulator",
		})
	}

	if config.TemplatesDirectory == "" || config.OutputDirectory == "" {
		payload.Warnings = append(payload.Warnings, msg(msgExportNotConfigured))
	}

	return payload
}
