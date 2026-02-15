package openkneeboard

import (
	"fmt"
	"simdiag/common"
)

// addBindings adds OpenKneeboard bindings to an ExportDevice
func addBindings(exportDevice *common.ExportDevice, config *common.Config) {
	// Check if OpenKneeboard is configured
	if config.OpenKneeboardProfilesFilepath == "" {
		return
	}

	// Load OpenKneeboard bindings for this device
	okneeboardBindings := LoadBindings(config.OpenKneeboardProfilesFilepath, exportDevice.Device.GUID)
	if len(okneeboardBindings) == 0 {
		return
	}

	// Convert OpenKneeboard bindings to regular bindings
	for _, okb := range okneeboardBindings {
		binding := common.Binding{
			// Use the source device GUID (from IL-2/DCS) instead of OpenKneeboard's GUID
			// This ensures bindings appear in the correct simulator's export with correct GUID
			DeviceGUID:  exportDevice.Device.GUID,
			DeviceName:  exportDevice.Device.Name,
			InputType:   okb.InputType,
			InputID:     okb.InputID,
			Action:      "OpenKneeboard",
			Description: "",
		}

		// Build description: "OpenKneeboard: ACTION"
		binding.Description = fmt.Sprintf("OpenKneeboard: %s", formatOpenKneeboardAction(okb.Action))

		exportDevice.Profile.Bindings = append(exportDevice.Profile.Bindings, binding)
	}
}

// formatOpenKneeboardAction formats an OpenKneeboard action for display
func formatOpenKneeboardAction(action string) string {
	// Convert SCREAMING_SNAKE_CASE to Title Case
	// Example: PREVIOUS_TAB -> Previous Tab
	words := make([]string, 0)
	currentWord := ""

	for _, char := range action {
		if char == '_' {
			if currentWord != "" {
				words = append(words, currentWord)
				currentWord = ""
			}
		} else {
			if currentWord == "" {
				currentWord = string(char)
			} else {
				currentWord += string(char + 32) // Convert to lowercase
			}
		}
	}

	if currentWord != "" {
		words = append(words, currentWord)
	}

	result := ""
	for i, word := range words {
		if i > 0 {
			result += " "
		}
		result += word
	}

	return result
}

// LoadBindingsForDevice loads OpenKneeboard bindings for a specific device using config
// Returns empty slice if OpenKneeboard is not configured or device has no bindings
func LoadBindingsForDevice(deviceGUID string, config *common.Config) []*Binding {
	if config == nil || config.OpenKneeboardProfilesFilepath == "" {
		return nil
	}

	return LoadBindings(config.OpenKneeboardProfilesFilepath, deviceGUID)
}
