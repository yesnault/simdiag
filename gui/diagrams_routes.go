package gui

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"simdiag/common"
	"simdiag/svg"
	"simdiag/workflow"
)

// openRequest asks the OS file manager to show a path inside the output
// directory: "." is the output directory itself, anything else a diagram in it.
type openRequest struct {
	Path string `json:"path"`
}

func registerDiagramRoutes(mux *http.ServeMux, state *State) {
	mux.HandleFunc("GET /api/diagrams", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, scanDiagrams(state.Config()))
	})

	mux.HandleFunc("POST /api/diagrams/open", func(w http.ResponseWriter, r *http.Request) {
		var req openRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Only ever open something inside the configured output directory.
		target, err := safeJoin(state.Config().OutputDirectory, req.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if _, err := os.Stat(target); err != nil {
			// Reported rather than silently doing nothing: on Windows explorer
			// swallows its own failures, so the click would look ignored.
			writeMessageError(w, http.StatusBadRequest, msgFolderMissing, map[string]string{"path": target})
			return
		}

		if err := openInFileManager(target); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]bool{"opened": true})
	})

	// The GUI equivalent of simdiag.exe -csv: rebuild the diagrams from the
	// CSV already on disk, without re-parsing the simulators.
	mux.HandleFunc("POST /api/diagrams/regenerate", func(w http.ResponseWriter, _ *http.Request) {
		config, err := state.ConfigSnapshot()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		csvPath := filepath.Join(config.OutputDirectory, "export.csv")
		if !fileExists(csvPath) {
			writeMessageError(w, http.StatusBadRequest, msgNoCSVToRegen, nil)
			return
		}

		ctx, cancel := context.WithCancel(context.Background())

		// Regeneration shares the export's run slot: both redirect the
		// process-wide progress writer.
		if !currentExport.begin(cancel) {
			cancel()
			writeMessageError(w, http.StatusConflict, msgExportRunning, nil)
			return
		}

		go runRegenerate(ctx, cancel, currentExport, config, csvPath)

		w.WriteHeader(http.StatusAccepted)
		writeJSON(w, currentExport.stateSince(0))
	})
}

// runRegenerate rebuilds the diagrams from an existing CSV.
func runRegenerate(ctx context.Context, cancel context.CancelFunc, run *exportRun, config *common.Config, csvPath string) {
	defer cancel()

	started := time.Now()

	writer := newLineWriter(run.appendLine)
	common.SetOutput(writer)
	defer common.SetOutput(nil)

	common.Printf("Regenerating diagrams from %s\n", csvPath)

	validationErrors, err := svg.GenerateSVGFromCSV(ctx, csvPath, config)
	writer.Flush()

	summary := &workflow.ExportSummary{
		CSVPath:          csvPath,
		ValidationErrors: validationErrors,
		DurationMS:       time.Since(started).Milliseconds(),
	}

	failure := ""
	if err != nil && ctx.Err() == nil {
		failure = err.Error()
	}

	run.finish(summary, failure)
}

// openInFileManager shows target to the user.
//
// A directory is opened; a file is revealed inside its own directory. Both are
// asked for: the toolbar button opens the output directory itself, while a
// group's button points at one of its diagrams to land in that module's folder.
func openInFileManager(target string) error {
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	isDir := info.IsDir()

	switch runtime.GOOS {
	case "windows":
		// explorer.exe exits with a non-zero status even when it succeeds, so
		// its error is deliberately ignored.
		if isDir {
			_ = exec.Command("explorer", target).Run()
		} else {
			_ = exec.Command("explorer", "/select,", target).Run()
		}
		return nil
	case "darwin":
		if isDir {
			return exec.Command("open", target).Run()
		}
		return exec.Command("open", "-R", target).Run()
	default:
		if isDir {
			return exec.Command("xdg-open", target).Run()
		}
		return exec.Command("xdg-open", filepath.Dir(target)).Run()
	}
}
