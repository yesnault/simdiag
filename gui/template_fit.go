package gui

import (
	"strings"

	"simdiag/common"
)

// How well a template fits a controller. It ranks the candidates the
// Devices tab offers, and the 0.5 threshold below is a presentation choice,
// which is why it lives here rather than in common.

// CheckCompatibility checks if a device is compatible with a template
func CheckCompatibility(device *common.Device, profile *common.Profile, template *common.Template) (compatible bool, score int, missing []string) {
	deviceInputs := collectDeviceInputs(device, profile)
	score = calculateMatchScore(deviceInputs, template)
	missing = findMissingInputs(deviceInputs, template)
	compatible = isCompatible(score, template)

	return compatible, score, missing
}

// collectDeviceInputs collects all input keys for a device from its bindings
func collectDeviceInputs(device *common.Device, profile *common.Profile) map[string]bool {
	deviceInputs := make(map[string]bool)

	for _, binding := range profile.Bindings {
		if binding.DeviceGUID != device.GUID {
			continue
		}

		key := formatInputKey(binding)
		if key != "" {
			deviceInputs[key] = true
		}
	}

	return deviceInputs
}

// formatInputKey renders a binding as a template key in the uppercase form
// LoadTemplate stores. It delegates to common.TemplateKeyFor so the two never drift:
// hand-rolling it here used to miss the "_OFF" suffix strip, which made every
// switch-off binding look absent from its template.
func formatInputKey(binding common.Binding) string {
	return strings.ToUpper(common.TemplateKeyFor(binding.InputType, binding.InputID))
}

// calculateMatchScore counts how many device inputs match template inputs
func calculateMatchScore(deviceInputs map[string]bool, template *common.Template) int {
	score := 0

	for _, templateButton := range template.Buttons {
		if deviceInputs[templateButton] {
			score++
		}
	}

	for _, templateAxis := range template.Axes {
		if deviceInputs[templateAxis] {
			score++
		}
	}

	for _, templateHat := range template.Hats {
		if deviceInputs[templateHat] {
			score++
		}
	}

	return score
}

// findMissingInputs finds device inputs that are not in the template
func findMissingInputs(deviceInputs map[string]bool, template *common.Template) []string {
	missing := make([]string, 0)

	for inputKey := range deviceInputs {
		if !isInputInTemplate(inputKey, template) {
			missing = append(missing, inputKey)
		}
	}

	return missing
}

// isInputInTemplate checks if an input key exists in the template
func isInputInTemplate(inputKey string, template *common.Template) bool {
	for _, templateKey := range template.Buttons {
		if templateKey == inputKey {
			return true
		}
	}

	for _, templateKey := range template.Axes {
		if templateKey == inputKey {
			return true
		}
	}

	for _, templateKey := range template.Hats {
		if templateKey == inputKey {
			return true
		}
	}

	return false
}

// isCompatible determines if the device is compatible based on match score
func isCompatible(score int, template *common.Template) bool {
	totalTemplateKeys := len(template.Buttons) + len(template.Axes) + len(template.Hats)
	if totalTemplateKeys == 0 {
		return true
	}

	compatibilityRatio := float64(score) / float64(totalTemplateKeys)
	return compatibilityRatio >= 0.5
}
