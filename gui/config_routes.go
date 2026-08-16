package gui

import (
	"encoding/json"
	"net/http"

	"github.com/wailsapp/wails/v3/pkg/application"

	"simdiag/templates"
)

// configPayload is what the Configuration tab loads and reloads.
type configPayload struct {
	ConfigPath string       `json:"configPath"`
	Config     configDTO    `json:"config"`
	Defaults   configDTO    `json:"defaults"`
	Status     configStatus `json:"status"`
}

func (s *State) configPayload() configPayload {
	config := s.Config()
	return configPayload{
		ConfigPath: s.ConfigPath(),
		Config:     toDTO(config),
		Defaults:   defaultsDTO(config),
		Status:     computeStatus(config),
	}
}

// registerConfigRoutes wires the Configuration tab.
func registerConfigRoutes(mux *http.ServeMux, state *State) {
	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, state.configPayload())
	})

	// Writing the form back is an explicit user action: the tab keeps an
	// unsaved-changes marker rather than persisting on every keystroke.
	mux.HandleFunc("PUT /api/config", func(w http.ResponseWriter, r *http.Request) {
		var dto configDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			http.Error(w, "invalid configuration payload: "+err.Error(), http.StatusBadRequest)
			return
		}

		applyDTO(state.Config(), dto)

		if err := state.Save(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, state.configPayload())
	})

	// SimDiag ships the templates of the common controllers inside the binary,
	// so a fresh install has something to draw with. Writing them is an explicit
	// action, and it never touches a file that is already there.
	mux.HandleFunc("POST /api/templates/install", func(w http.ResponseWriter, _ *http.Request) {
		// Same guard as a configuration switch: this writes under a root a
		// running export is reading from.
		if currentExport.isRunning() {
			writeMessageError(w, http.StatusConflict, msgExportRunning, nil)
			return
		}

		config := state.Config()
		target := config.TemplatesDirectory
		if target == "" {
			target = defaultTemplatesDirectory
		}

		written, err := templates.Install(target)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Files without the setting would leave the user exactly as stuck, so
		// record where they went.
		if config.TemplatesDirectory == "" {
			config.TemplatesDirectory = target
			if err := state.Save(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		writeJSON(w, installedPayload{
			Installed: len(written),
			Target:    target,
			Config:    state.configPayload(),
		})
	})

	mux.HandleFunc("POST /api/browse", handleBrowse)
}

// installedPayload reports what the install wrote, with the refreshed form so
// the tab can redraw without asking again.
type installedPayload struct {
	Installed int           `json:"installed"`
	Target    string        `json:"target"`
	Config    configPayload `json:"config"`
}

// browseRequest asks for a native file or folder picker.
type browseRequest struct {
	Kind    string `json:"kind"`  // "folder" or "file"
	Title   string `json:"title"` // dialog caption
	Current string `json:"current"`
	Filter  string `json:"filter"` // e.g. "*.exe" or "*.json"
}

// handleBrowse opens the OS picker and returns the chosen path.
//
// Dialogs must run on the UI thread, and this executes on an HTTP goroutine, so
// the call is marshalled across with InvokeSyncWithResultAndError.
func handleBrowse(w http.ResponseWriter, r *http.Request) {
	var req browseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid browse request: "+err.Error(), http.StatusBadRequest)
		return
	}

	path, err := application.InvokeSyncWithResultAndError(func() (string, error) {
		dialog := application.Get().Dialog.OpenFile()
		dialog.SetTitle(req.Title)
		dialog.CanChooseDirectories(req.Kind == "folder")
		dialog.CanChooseFiles(req.Kind != "folder")

		if dir := existingDirectory(req.Current); dir != "" {
			dialog.SetDirectory(dir)
		}
		if req.Filter != "" {
			dialog.AddFilter(req.Filter, req.Filter)
		}

		return dialog.PromptForSingleSelection()
	})
	if dialogCancelled(err) {
		path, err = "", nil
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// An empty path means the user cancelled; the frontend leaves the field alone.
	writeJSON(w, map[string]string{"path": path})
}
