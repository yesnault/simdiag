package svg

import (
	"fmt"
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

		// Build the expected key based on input type
		// Strip _OFF suffix so BTN25_OFF validates against Button_25 (not Button_25_OFF)
		templateInputID := strings.TrimSuffix(binding.InputID, "_OFF")

		var expectedKey string
		switch binding.InputType {
		case common.Button:
			expectedKey = fmt.Sprintf("Button_%s", templateInputID)
		case common.Axis:
			expectedKey = fmt.Sprintf("AXIS_%s", strings.ToUpper(templateInputID))
		case common.Hat:
			expectedKey = fmt.Sprintf("POV_%s", strings.ToUpper(templateInputID))
		}

		// Filter out TARGET virtual buttons (beyond physical device limits)
		// Only for Thrustmaster HOTAS Warthog when using TARGET profiles
		// Thrustmaster HOTAS Warthog Throttle: max 33 physical buttons
		// Thrustmaster HOTAS Warthog Joystick: max 19 physical buttons
		if binding.InputType == common.Button && exportDevice.Profile.SimType == "TARGET" {
			buttonNum := 0
			_, _ = fmt.Sscanf(binding.InputID, "%d", &buttonNum)

			// Check if this is a Thrustmaster HOTAS Warthog device
			deviceNameLower := strings.ToLower(exportDevice.Device.Name)
			isWarthog := strings.Contains(deviceNameLower, "warthog")

			if isWarthog {
				isThrottle := strings.Contains(deviceNameLower, "throttle")
				isJoystick := strings.Contains(deviceNameLower, "joystick")

				// Skip virtual buttons beyond physical limits
				if (isThrottle && buttonNum > 33) || (isJoystick && buttonNum > 19) {
					continue // Skip this virtual button, don't report as error
				}
			}
		}

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

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("⚠ %d VALIDATION ERROR(S) FOUND\n", len(errors))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("\nThe following bindings were found in the simulator configuration")
	fmt.Println("but have NO corresponding key in the SVG template:")
	fmt.Println()

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
		fmt.Printf("Device: %s (%d error(s)) - %s\n", deviceKey.DeviceName, len(deviceErrors), deviceKey.SimType)
		for _, err := range deviceErrors {
			fmt.Printf("  ✗ %s %s → %s\n", err.InputType, err.InputID, err.Action)
			fmt.Printf("    Missing template key: %s\n", err.MissingKey)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("These bindings will NOT appear in the generated diagrams.")
	fmt.Println("Update your SVG template to include the missing keys.")
	fmt.Println(strings.Repeat("=", 80))
}
