package gui

import (
	"encoding/json"
	"fmt"
	"net/http"

	"simdiag/common"
	"simdiag/target"
)

// registerDeviceRoutes wires the Devices tab.
func registerDeviceRoutes(mux *http.ServeMux, state *State) {
	// Scanning re-parses every simulator, so it is an explicit action rather
	// than something that happens on every tab switch.
	mux.HandleFunc("GET /api/devices", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, scanDevices(state.Config()))
	})

	mux.HandleFunc("POST /api/devices/mapping", func(w http.ResponseWriter, r *http.Request) {
		var req mappingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.GUID == "" {
			http.Error(w, "device guid is required", http.StatusBadRequest)
			return
		}

		config := state.Config()

		switch req.Action {
		case "assign":
			if req.TemplatePath == "" {
				http.Error(w, "template path is required to assign", http.StatusBadRequest)
				return
			}
			// Confirm the template exists and stays inside the templates
			// directory before writing it into the configuration.
			if _, err := templateAbsolutePath(config, req.TemplatePath); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			config.UpdateDeviceMapping(req.GUID, req.Name, req.TemplatePath, config.TemplatesDirectory)

		case "skip":
			config.MarkDeviceAsSkipped(req.GUID, req.Name)

		default:
			http.Error(w, fmt.Sprintf("unknown action %q", req.Action), http.StatusBadRequest)
			return
		}

		if err := state.Save(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, scanDevices(config))
	})

	mux.HandleFunc("POST /api/devices/target", func(w http.ResponseWriter, r *http.Request) {
		var req targetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}

		config := state.Config()
		config.UpdateDeviceTargetNumber(req.GUID, req.Name, req.TargetNumber)

		if err := state.Save(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, scanDevices(config))
	})

	// TARGET numbers can be guessed from the profile, which beats asking the user
	// to know that a Warthog stick is device 1001.
	mux.HandleFunc("POST /api/devices/target/detect", func(w http.ResponseWriter, _ *http.Request) {
		config := state.Config()

		matched, err := detectTargetNumbers(config)
		if err != nil {
			writeMessageError(w, http.StatusBadRequest, msgTargetDetectFailed, errorArg(err))
			return
		}

		if matched > 0 {
			if err := state.Save(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		writeJSON(w, detectPayload{Matched: matched, Devices: scanDevices(config)})
	})

	// There is deliberately no route for DCS modules. They are detected from the
	// configured path, and the Gremlins and TARGET profiles in the Configuration
	// tab apply to all of them, so there is nothing per-module left to record.
}

// detectTargetNumbers matches the device numbers a TARGET profile uses against
// the physical controllers, by name, and records what it finds.
//
// The command line used to do this during its configuration walk; it is the one
// thing that walk did which the interface could not.
func detectTargetNumbers(config *common.Config) (int, error) {
	profilePath := targetProfilePath(config)
	if profilePath == "" {
		return 0, fmt.Errorf("no TARGET profile is configured")
	}

	numbers, err := target.GetTargetDeviceNumbers(profilePath)
	if err != nil {
		return 0, err
	}
	if len(numbers) == 0 {
		return 0, nil
	}

	found, _ := parseControllers(config, &devicesPayload{})
	devices := make([]*common.Device, 0, len(found))
	for _, s := range found {
		devices = append(devices, s.device)
	}

	// Auto-matching is by controller name, so a virtual device would match on the
	// name it borrows from the physical one it stands for.
	mappings := target.AutoMatchTargetDevices(numbers, common.FilterPhysicalDevices(devices))
	for _, mapping := range mappings {
		config.UpdateDeviceTargetNumber(mapping.DeviceGUID, mapping.DeviceName, mapping.DeviceNumber)
	}

	return len(mappings), nil
}

// targetProfilePath returns the first TARGET profile any simulator declares.
// They are usually the same file: one profile serves the whole rig.
func targetProfilePath(config *common.Config) string {
	for _, simType := range simulatorOrder {
		if path := target.GetProfilePath(config, simType); path != "" {
			return path
		}
	}
	return ""
}

// detectPayload reports how many controllers were matched, alongside the redrawn
// device list, so one round trip instead of two.
type detectPayload struct {
	Matched int            `json:"matched"`
	Devices devicesPayload `json:"devices"`
}

// mappingRequest assigns a template to a device, or marks it as ignored.
type mappingRequest struct {
	Action       string `json:"action"` // "assign" or "skip"
	GUID         string `json:"guid"`
	Name         string `json:"name"`
	TemplatePath string `json:"templatePath"`
}

// targetRequest sets the Thrustmaster TARGET device number (1001, 1002, ...).
type targetRequest struct {
	GUID string `json:"guid"`
	// Name is carried so a controller with no mapping yet can be given one
	// rather than having the number silently dropped.
	Name         string `json:"name"`
	TargetNumber int    `json:"targetNumber"`
}
