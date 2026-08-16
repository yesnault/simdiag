package workflow

import (
	"os"

	"simdiag/common"
	"simdiag/gremlins"
	"simdiag/openkneeboard"
)

// Pairing devices with their templates lives here rather than in common because
// it has to ask Gremlins and OpenKneeboard whether they know a controller the
// simulator never mentioned. common cannot import those packages — they import
// it — so it used to reach them through a function-pointer global populated at
// startup (common.ExtFuncs, app.Wire). workflow has no such constraint and calls
// them directly, which is what let the indirection go.

// loadDeviceTemplatePaths resolves each device's template without reading the
// SVG. The CSV only needs the path; the SVG generator loads the file later.
func loadDeviceTemplatePaths(devices map[string]*common.Device, config *common.Config) map[string]*common.Template {
	addMissingDevicesWithExternalBindings(devices, config)

	deviceTemplates := make(map[string]*common.Template)

	for guid, device := range devices {
		if device.IsVirtual {
			continue
		}

		mapping := config.GetTemplateMappingForDevice(guid)
		if mapping != nil && mapping.SkipTemplate {
			continue
		}

		templatePath := templatePathFor(mapping, config)
		if templatePath == "" {
			continue
		}

		deviceTemplates[guid] = &common.Template{
			FilePath: templatePath,
			Name:     common.GetTemplateNameFromPath(templatePath),
		}
	}

	return deviceTemplates
}

// addMissingDevicesWithExternalBindings adds the controllers that only an
// external tool knows about.
//
// A device remapped through Gremlins may never appear in the simulator's own
// configuration, and it still deserves a diagram.
func addMissingDevicesWithExternalBindings(devices map[string]*common.Device, config *common.Config) {
	if config == nil {
		return
	}

	for _, mapping := range config.DeviceMappings {
		if _, exists := devices[mapping.DeviceGUID]; exists {
			continue
		}

		// A partial match means the same controller is already here under the
		// other simulator's GUID format; adding it again would duplicate it.
		if matchesKnownDevice(mapping.DeviceGUID, devices) {
			continue
		}

		if hasExternalBindings(mapping.DeviceGUID, config) {
			devices[mapping.DeviceGUID] = &common.Device{
				GUID: mapping.DeviceGUID,
				Name: mapping.DeviceName,
			}
		}
	}
}

func matchesKnownDevice(guid string, devices map[string]*common.Device) bool {
	for existingGUID := range devices {
		if common.MatchGUIDPartial(guid, existingGUID) {
			return true
		}
	}
	return false
}

func hasExternalBindings(guid string, config *common.Config) bool {
	if config == nil {
		return false
	}

	return len(gremlins.LoadBindingsForDevice(guid, config)) > 0 ||
		len(openkneeboard.LoadBindingsForDevice(guid, config)) > 0
}

// templatePathFor resolves a mapping to an absolute template path.
//
// A mapping naming a template that is no longer on disk is reported rather than
// dropped. Renaming or deleting a template used to make its controller lose its
// diagram silently: the export said nothing and the CSV came back with an empty
// Template column, which reads like the device was never configured.
func templatePathFor(mapping *common.DeviceTemplateMapping, config *common.Config) string {
	if mapping == nil || mapping.TemplateFilepath == "" {
		return ""
	}

	absolutePath := common.MakeAbsolutePath(mapping.TemplateFilepath, config.TemplatesDirectory)
	if _, err := os.Stat(absolutePath); err == nil {
		return absolutePath
	}

	common.Printf("  ⚠ %s: template %s is not there any more, the device is exported without one\n",
		mapping.DeviceName, mapping.TemplateFilepath)

	return ""
}
