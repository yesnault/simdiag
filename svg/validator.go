package svg

import (
	"simdiag/common"
	"strings"
)

// ValidateBindings checks if all bindings have corresponding keys in the template
func ValidateBindings(exportDevice *common.ExportDevice) []common.ValidationError {
	var errors []common.ValidationError

	if exportDevice.Template == nil || exportDevice.Profile == nil {
		return errors
	}

	// Build a set of available keys in the template
	availableKeys := make(map[string]bool)

	// Add all buttons
	for _, btn := range exportDevice.Template.Buttons {
		availableKeys[btn] = true
	}

	// Add all axes
	for _, axis := range exportDevice.Template.Axes {
		availableKeys[axis] = true
	}

	// Add all hats
	for _, hat := range exportDevice.Template.Hats {
		availableKeys[hat] = true
	}

	// Check each binding
	deviceID := common.NormalizeGUIDShort(exportDevice.Device.GUID)

	for _, binding := range exportDevice.Profile.Bindings {
		// Skip bindings for other devices (compare using first 8 characters)
		bindingID := common.NormalizeGUIDShort(binding.DeviceGUID)

		if bindingID != deviceID {
			continue
		}

		// The same function the generator uses, so the validator cannot decide a
		// binding is missing from a key the generator would have filled in. It
		// was a second copy of the switch until it was not.
		expectedKey := common.TemplateKeyFor(binding.InputType, binding.InputID)

		// There used to be a block here skipping Warthog buttons past the
		// hardware's physical count, gated on SimType == "TARGET". No parser
		// ever sets that: SimType only ever holds one of the three simulator
		// constants, so the block never ran. It went with the hardware limits it
		// hardcoded, which were configuration rather than validation anyway.

		// Normalize key to uppercase for comparison
		normalizedKey := strings.ToUpper(expectedKey)

		// Check if key exists in template (case-insensitive)
		if !availableKeys[normalizedKey] {
			errors = append(errors, common.ValidationError{
				DeviceName: exportDevice.Device.Name,
				SimType:    exportDevice.Profile.SimType,
				InputType:  binding.InputType,
				InputID:    binding.InputID,
				Action:     binding.Action,
				MissingKey: expectedKey,
			})
		}
	}

	return errors
}

// DisplayValidationErrors displays all validation errors found during export
func DisplayValidationErrors(errors []common.ValidationError) {
	if len(errors) == 0 {
		return
	}

	common.Println("\n" + strings.Repeat("=", 80))
	common.Printf("⚠ %d VALIDATION ERROR(S) FOUND\n", len(errors))
	common.Println(strings.Repeat("=", 80))
	common.Println("\nThe following bindings were found in the simulator configuration")
	common.Println("but have NO corresponding key in the SVG template:")
	common.Println()

	// Group errors by device and simulator
	type DeviceKey struct {
		DeviceName string
		SimType    common.SimulationType
	}
	errorsByDevice := make(map[DeviceKey][]common.ValidationError)
	for _, err := range errors {
		key := DeviceKey{DeviceName: err.DeviceName, SimType: err.SimType}
		errorsByDevice[key] = append(errorsByDevice[key], err)
	}

	// Display errors grouped by device
	for deviceKey, deviceErrors := range errorsByDevice {
		common.Printf("Device: %s (%d error(s)) - %s\n", deviceKey.DeviceName, len(deviceErrors), deviceKey.SimType)
		for _, err := range deviceErrors {
			common.Printf("  ✗ %s %s → %s\n", err.InputType, err.InputID, err.Action)
			common.Printf("    Missing template key: %s\n", err.MissingKey)
		}
		common.Println()
	}

	common.Println(strings.Repeat("=", 80))
	common.Println("These bindings will NOT appear in the generated diagrams.")
	common.Println("Update your SVG template to include the missing keys.")
	common.Println(strings.Repeat("=", 80))
}
