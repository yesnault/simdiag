package common

import (
	"os"
)

// GroupDevicesByTemplate groups devices by their template filepath
// Returns a map: template filepath -> list of device GUIDs
func GroupDevicesByTemplate(deviceTemplates map[string]*Template) map[string][]string {
	templateGroups := make(map[string][]string)

	for deviceGUID, template := range deviceTemplates {
		if template == nil {
			continue
		}

		// Use template filepath as the key for grouping
		templateKey := template.FilePath
		templateGroups[templateKey] = append(templateGroups[templateKey], deviceGUID)
	}

	return templateGroups
}

// LoadDeviceTemplatePathsOnly loads only template paths (metadata) without loading SVG content
// Used in CSV-only mode to populate template paths in CSV without validation
func LoadDeviceTemplatePathsOnly(devices map[string]*Device, config *Config) map[string]*Template {
	// Add missing devices that have external bindings
	addMissingDevicesWithExternalBindings(devices, config)

	deviceTemplates := make(map[string]*Template)

	for guid, device := range devices {
		if device.IsVirtual {
			continue
		}

		mapping := config.GetTemplateMappingForDevice(guid)
		if mapping != nil && mapping.SkipTemplate {
			continue
		}

		// Get template path from config
		templatePath := getTemplatePath(mapping, config)
		if templatePath == "" {
			continue
		}

		// Create lightweight template with only path metadata (no SVG content)
		// Name is extracted from filepath (without .svg extension)
		templateName := GetTemplateNameFromPath(templatePath)
		deviceTemplates[guid] = &Template{
			FilePath: templatePath,
			Name:     templateName,
		}
	}

	return deviceTemplates
}

// addMissingDevicesWithExternalBindings adds devices with external bindings to the devices map
func addMissingDevicesWithExternalBindings(devices map[string]*Device, config *Config) {
	if config == nil {
		return
	}

	for _, mapping := range config.DeviceMappings {
		// Check for exact match first
		if _, exists := devices[mapping.DeviceGUID]; exists {
			continue
		}

		// Check if a device with a partially matching GUID already exists (IL-2 vs DCS format)
		// If so, skip adding this mapping to avoid duplicates
		hasPartialMatch := false
		for existingGUID := range devices {
			if MatchGUIDPartial(mapping.DeviceGUID, existingGUID) {
				hasPartialMatch = true
				break
			}
		}
		if hasPartialMatch {
			continue
		}

		if hasExternalBindingsForGUID(mapping.DeviceGUID, config) {
			devices[mapping.DeviceGUID] = &Device{
				GUID: mapping.DeviceGUID,
				Name: mapping.DeviceName,
			}
		}
	}
}

// hasExternalBindingsForGUID checks if a device GUID has external bindings
func hasExternalBindingsForGUID(guid string, config *Config) bool {
	if config == nil || ExtFuncs == nil {
		return false
	}

	if gremlinsBindings := ExtFuncs.LoadGremlinsBindingsForDevice(guid, config); gremlinsBindings != nil {
		return true
	}

	if openKneeboardBindings := ExtFuncs.LoadOpenKneeboardBindingsForDevice(guid, config); openKneeboardBindings != nil {
		return true
	}

	return false
}

// loadDeviceTemplate loads a template for a single device
// getTemplatePath gets the absolute path to a template from mapping
func getTemplatePath(mapping *DeviceTemplateMapping, config *Config) string {
	if mapping == nil {
		return ""
	}

	absolutePath := MakeAbsolutePath(mapping.TemplateFilepath, config.TemplatesDirectory)
	if _, err := os.Stat(absolutePath); err == nil {
		return absolutePath
	}

	return ""
}
