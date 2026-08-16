package gui

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"simdiag/common"
)

// configFilter is what the pickers show. Windows separates patterns with ";".
const configFilter = "*.yaml;*.yml"

// sessionPayload is what the header shows: which configuration is open, and
// which ones to offer as shortcuts.
type sessionPayload struct {
	Version    string        `json:"version"`
	ConfigPath string        `json:"configPath"`
	Recent     []recentEntry `json:"recent"`
	// Language is the interface language the user picked, empty until they pick
	// one. The frontend then follows the browser, which reports Windows' own
	// language. Go never renders anything with this: it stores it and hands it
	// back, the one exception being the native OS dialogs (see dialogText).
	Language string `json:"language"`
	// Cancelled reports a picker the user dismissed. The session is then the
	// unchanged current one, and the frontend leaves everything alone.
	Cancelled bool `json:"cancelled,omitempty"`
}

func (s *State) sessionPayload() sessionPayload {
	return sessionPayload{
		Version:    s.Version(),
		ConfigPath: s.ConfigPath(),
		// Always a list, never null: the frontend reads its length first.
		Recent:   loadRecent(),
		Language: loadSettings().Language,
	}
}

// registerSessionRoutes wires the configuration picker in the header.
func registerSessionRoutes(mux *http.ServeMux, state *State) {
	mux.HandleFunc("GET /api/session", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, state.sessionPayload())
	})

	mux.HandleFunc("POST /api/config/open", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path string `json:"path"`
		}
		// An empty body means "ask the user", so end-of-input is not an error.
		// Do not gate this on Content-Length: requests reaching the Wails asset
		// server do not always carry one, and skipping the decode then silently
		// turns "open this recent file" into "open the file dialog".
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}

		if !allowSwitch(w) {
			return
		}

		path := req.Path
		if path == "" {
			chosen, err := pickConfigToOpen(state.ConfigPath())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if chosen == "" {
				writeCancelled(w, state)
				return
			}
			path = chosen
		}

		switchConfig(w, state, path)
	})

	mux.HandleFunc("POST /api/config/new", func(w http.ResponseWriter, _ *http.Request) {
		if !allowSwitch(w) {
			return
		}

		path, err := pickConfigToCreate(state.ConfigPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if path == "" {
			writeCancelled(w, state)
			return
		}

		// An existing file is opened, never overwritten: a mapping_config.yaml
		// is a profile someone built up over hours, and the visible outcome (that
		// file becomes the current one) is the same either way.
		if !fileExists(path) {
			if err := common.SaveConfigTo(&common.Config{}, path); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		switchConfig(w, state, path)
	})

	// The language is a user preference, not part of the configuration: it is
	// stored beside the recent-files list and survives switching profiles.
	mux.HandleFunc("POST /api/language", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Language string `json:"language"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if !isSupportedLanguage(req.Language) {
			http.Error(w, "unsupported language: "+req.Language, http.StatusBadRequest)
			return
		}

		if err := updateSettings(func(s *settings) { s.Language = req.Language }); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, state.sessionPayload())
	})

	// Rereading the file from disk is the same operation as opening it, and it
	// is how a user picks up an edit made in a text editor.
	mux.HandleFunc("POST /api/config/reload", func(w http.ResponseWriter, _ *http.Request) {
		if !allowSwitch(w) {
			return
		}
		switchConfig(w, state, state.ConfigPath())
	})
}

// allowSwitch refuses to change configuration while an export is running.
//
// A run holds a snapshot of the old configuration and reads it for a minute or
// more; switching moves the working directory out from under it, so every
// relative path it has yet to resolve would point somewhere else.
func allowSwitch(w http.ResponseWriter) bool {
	if currentExport.isRunning() {
		writeMessageError(w, http.StatusConflict, msgExportRunning, nil)
		return false
	}
	return true
}

func switchConfig(w http.ResponseWriter, state *State, path string) {
	if err := state.SwitchTo(path); err != nil {
		if errors.Is(err, errNoConfigFile) {
			writeMessageError(w, http.StatusBadRequest, msgNoConfigFile, map[string]string{"path": path})
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rememberRecent(state.ConfigPath())
	writeJSON(w, state.sessionPayload())
}

func writeCancelled(w http.ResponseWriter, state *State) {
	payload := state.sessionPayload()
	payload.Cancelled = true
	writeJSON(w, payload)
}

// dialogCancelled reports whether an error is really the user dismissing the
// picker.
//
// The Windows common file dialog reports a cancellation as an error rather than
// as an empty selection, and its sentinel (cfd.ErrorCancelled) lives in a Wails
// internal/ package that cannot be imported, so the message is all there is to
// match on. Getting this wrong shows the user a red failure for pressing Cancel.
func dialogCancelled(err error) bool {
	return err != nil && strings.Contains(err.Error(), "cancelled by user")
}

// pickConfigToOpen asks for an existing configuration file. An empty result
// means the user dismissed the dialog.
//
// Dialogs must run on the UI thread and this executes on an HTTP goroutine, so
// the call is marshalled across, exactly as handleBrowse does.
func pickConfigToOpen(current string) (string, error) {
	path, err := application.InvokeSyncWithResultAndError(func() (string, error) {
		dialog := application.Get().Dialog.OpenFile()
		dialog.SetTitle(dialogText("dialog.open.title"))
		dialog.CanChooseFiles(true)
		dialog.CanChooseDirectories(false)
		dialog.AddFilter(dialogText("dialog.filter")+" ("+configFilter+")", configFilter)

		if dir := existingDirectory(current); dir != "" {
			dialog.SetDirectory(dir)
		}

		return dialog.PromptForSingleSelection()
	})
	if dialogCancelled(err) {
		return "", nil
	}

	return path, err
}

// pickConfigToCreate asks where a new configuration should live.
func pickConfigToCreate(current string) (string, error) {
	path, err := application.InvokeSyncWithResultAndError(func() (string, error) {
		dialog := application.Get().Dialog.SaveFile()
		dialog.SetMessage(dialogText("dialog.new.title"))
		dialog.SetFilename(configFileName)
		dialog.CanCreateDirectories(true)
		dialog.AddFilter(dialogText("dialog.filter")+" ("+configFilter+")", configFilter)

		if dir := existingDirectory(current); dir != "" {
			dialog.SetDirectory(dir)
		}

		return dialog.PromptForSingleSelection()
	})
	if dialogCancelled(err) {
		return "", nil
	}
	if err != nil || path == "" {
		return "", err
	}

	// The save dialog hands back whatever was typed, extension included or not.
	if filepath.Ext(path) == "" {
		path += ".yaml"
	}

	return path, nil
}
