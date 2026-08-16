package gui

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"

	"simdiag/app"
	"simdiag/common"
)

// templateOption is one candidate template for a device, already scored against
// that device's actual bindings.
type templateOption struct {
	Name       string   `json:"name"`
	Path       string   `json:"path"` // relative to the templates directory
	Buttons    int      `json:"buttons"`
	Axes       int      `json:"axes"`
	Hats       int      `json:"hats"`
	Score      int      `json:"score"` // device inputs the template can display
	Total      int      `json:"total"` // keys the template offers
	Compatible bool     `json:"compatible"`
	Missing    []string `json:"missing,omitempty"` // device inputs with no key
}

// density is the share of the template's own keys that this device fills. It
// separates a template built for this controller from a much larger one that
// happens to cover the same inputs.
func (o templateOption) density() float64 {
	if o.Total == 0 {
		return 0
	}
	return float64(o.Score) / float64(o.Total)
}

// deviceEntry is one physical controller, merged across simulators.
type deviceEntry struct {
	GUID         string           `json:"guid"`
	Name         string           `json:"name"`
	AllGUIDs     []string         `json:"allGuids"`
	Simulators   []string         `json:"simulators"`
	IsVirtual    bool             `json:"isVirtual"`
	Bindings     int              `json:"bindings"`
	TemplatePath string           `json:"templatePath"`
	TemplateName string           `json:"templateName"`
	Skipped      bool             `json:"skipped"`
	TargetNumber int              `json:"targetNumber"`
	Templates    []templateOption `json:"templates"`
}

// devicesPayload is what the Devices tab renders.
//
// It carries no DCS module list: the tab is about controllers, and the detected
// aircraft are shown where they are relevant: under the DCS path in the
// Configuration tab, and as targets in the Generate tab.
type devicesPayload struct {
	Devices       []deviceEntry `json:"devices"`
	TemplateCount int           `json:"templateCount"`
	Warnings      []message     `json:"warnings"`
}

// parseControllers reads every configured simulator and returns one entry per
// physical controller, merged across simulators, plus a profile holding all the
// bindings found, so a device can be scored whichever simulator it came from.
//
// It is separate from scanDevices because the TARGET auto-detection needs the
// controllers without the template ranking that goes with them.
func parseControllers(config *common.Config, payload *devicesPayload) ([]*scannedDevice, *common.Profile) {
	merged := &common.Profile{Bindings: []common.Binding{}}
	var found []*scannedDevice

	parsers := app.Parsers(config)
	for _, simType := range simulatorOrder {
		parser := parsers[simType]
		if parser == nil {
			continue
		}

		section := config.LookupSimulatorConfig(simType)
		if section == nil {
			continue
		}
		configPath := section.DCSPath
		if simType != common.DCSWorld {
			configPath = section.IL2InputPath
		}
		if configPath == "" {
			continue
		}

		profiles, err := parser.Parse(configPath)
		if err != nil {
			payload.Warnings = append(payload.Warnings, msgArgs(msgParserFailed,
				map[string]string{"parser": parser.GetName(), "error": err.Error()}))
			continue
		}

		for _, profile := range profiles.Profiles {
			merged.Bindings = append(merged.Bindings, profile.Bindings...)
		}

		for _, device := range common.GetAllDevicesFromProfiles(profiles) {
			// Same physical controller across simulators: DCS uses 5-segment
			// GUIDs and IL-2 four, so exact matching is not enough.
			if existing := findScannedDevice(found, device.GUID); existing != nil {
				if !slices.Contains(existing.guids, device.GUID) {
					existing.guids = append(existing.guids, device.GUID)
				}
				if !slices.Contains(existing.simulators, string(simType)) {
					existing.simulators = append(existing.simulators, string(simType))
				}
				continue
			}

			found = append(found, &scannedDevice{
				device:     device,
				guids:      []string{device.GUID},
				simulators: []string{string(simType)},
			})
		}
	}

	return found, merged
}

// scanDevices parses every configured simulator and returns one row per physical
// controller, each with its templates ranked by how well they fit that device's
// bindings. This replaces the prompt cascade the command line used to run.
func scanDevices(config *common.Config) devicesPayload {
	// Devices must encode as [] rather than null: the frontend reads its length
	// before anything else.
	payload := devicesPayload{Devices: []deviceEntry{}}

	found, merged := parseControllers(config, &payload)

	templates := loadTemplateCatalogue(config, &payload)
	payload.TemplateCount = len(templates)

	for _, s := range found {
		payload.Devices = append(payload.Devices, buildDeviceEntry(config, s.device, s.guids, s.simulators, merged, templates))
	}

	// Physical devices first, then virtual ones, alphabetically within each group
	// in the order the CLI shows.
	slices.SortFunc(payload.Devices, func(a, b deviceEntry) int {
		if a.IsVirtual != b.IsVirtual {
			if a.IsVirtual {
				return 1
			}
			return -1
		}
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return payload
}

// scannedDevice accumulates one physical controller as it is seen across
// simulators, each of which names it with its own GUID format.
type scannedDevice struct {
	device     *common.Device
	guids      []string
	simulators []string
}

// findScannedDevice returns the entry already holding this controller, matching
// on the first three GUID segments so a DCS 5-segment GUID and the IL-2
// 4-segment one for the same hardware land on the same row.
func findScannedDevice(entries []*scannedDevice, guid string) *scannedDevice {
	for _, entry := range entries {
		for _, known := range entry.guids {
			if common.MatchGUIDPartial(known, guid) {
				return entry
			}
		}
	}
	return nil
}

// loadTemplateCatalogue reads every template once, so each device can be scored
// against all of them without re-reading the files.
func loadTemplateCatalogue(config *common.Config, payload *devicesPayload) []*common.Template {
	if config.TemplatesDirectory == "" {
		payload.Warnings = append(payload.Warnings, msg(msgTemplatesUnconfigured))
		return nil
	}

	templates, err := common.FindTemplates(config.TemplatesDirectory)
	if err != nil {
		payload.Warnings = append(payload.Warnings, msgArgs(msgTemplatesReadFailed, errorArg(err)))
		return nil
	}
	return templates
}

// buildDeviceEntry assembles one row: its current mapping plus every template
// ranked against the bindings this device actually uses.
func buildDeviceEntry(config *common.Config, device *common.Device, guids, simulators []string, merged *common.Profile, templates []*common.Template) deviceEntry {
	entry := deviceEntry{
		GUID:       device.GUID,
		Name:       device.Name,
		AllGUIDs:   guids,
		Simulators: simulators,
		IsVirtual:  device.IsVirtual,
	}

	for _, binding := range merged.Bindings {
		if slices.Contains(guids, binding.DeviceGUID) {
			entry.Bindings++
		}
	}

	if mapping := config.GetTemplateMappingForDevice(device.GUID); mapping != nil {
		entry.TemplatePath = mapping.TemplateFilepath
		entry.TemplateName = common.GetTemplateNameFromPath(mapping.TemplateFilepath)
		entry.Skipped = mapping.SkipTemplate
		entry.TargetNumber = mapping.DeviceTargetNumber
	}

	for _, template := range templates {
		option := templateOption{
			Name:    template.Name,
			Path:    common.MakeRelativePath(template.FilePath, config.TemplatesDirectory),
			Buttons: len(template.Buttons),
			Axes:    len(template.Axes),
			Hats:    len(template.Hats),
			Total:   len(template.Buttons) + len(template.Axes) + len(template.Hats),
		}

		// A device may appear under several GUIDs; score each and keep the best,
		// since bindings are recorded per simulator-native GUID.
		for _, guid := range guids {
			compatible, score, missing := CheckCompatibility(&common.Device{GUID: guid, Name: device.Name}, merged, template)
			if score >= option.Score {
				option.Score = score
				option.Compatible = compatible
				option.Missing = missing
			}
		}

		entry.Templates = append(entry.Templates, option)
	}

	// Best fit first. Raw score alone is a bad ranking: a 109-key throttle
	// template incidentally covers as many of a button box's inputs as the
	// 42-key template actually made for it. So among templates that display the
	// same number of the device's inputs, prefer the one whose own keys are most
	// fully used: the tighter fit is the one the user means.
	slices.SortFunc(entry.Templates, func(a, b templateOption) int {
		if a.Score != b.Score {
			return b.Score - a.Score
		}
		if d := cmp.Compare(b.density(), a.density()); d != 0 {
			return d
		}
		if len(a.Missing) != len(b.Missing) {
			return len(a.Missing) - len(b.Missing)
		}
		return strings.Compare(a.Name, b.Name)
	})

	return entry
}

// templateAbsolutePath resolves a template path coming from the frontend, which
// is always relative to the configured templates directory.
func templateAbsolutePath(config *common.Config, relative string) (string, error) {
	return safeJoin(config.TemplatesDirectory, filepath.ToSlash(relative))
}
