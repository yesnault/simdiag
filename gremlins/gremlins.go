package gremlins

import (
	"fmt"
	"simdiag/common"
	"strings"
)

// buildVJoyGUIDMap builds a map of vJoy device numbers to GUID prefixes
// It tries to extract vJoy GUIDs from the configuration first, then falls back to hardcoded values
func buildVJoyGUIDMap(config *common.Config) map[int]string {
	vJoyGUIDs := make(map[int]string)

	if config == nil {
		// Fallback to hardcoded values if no config
		return map[int]string{
			1: "a768ef40-b302-11ea",
			2: "ed312110-b310-11ea",
		}
	}

	// Get device mappings from global config
	deviceMappings := config.DeviceMappings

	// Extract vJoy device GUIDs from device mappings
	// We'll use the order they appear in the config to assign device numbers
	vJoyNumber := 1
	for _, mapping := range deviceMappings {
		// Check if this is a vJoy device (case-insensitive)
		if strings.Contains(strings.ToLower(mapping.DeviceName), "vjoy") {
			// Extract GUID prefix (first 18 characters: "xxxxxxxx-xxxx-xxxx")
			guid := strings.ToLower(mapping.DeviceGUID)
			parts := strings.Split(guid, "-")
			if len(parts) >= 3 {
				guidPrefix := strings.Join(parts[:3], "-")
				vJoyGUIDs[vJoyNumber] = guidPrefix
				vJoyNumber++
			}
		}
	}

	// If no vJoy devices found in config, use hardcoded fallback
	if len(vJoyGUIDs) == 0 {
		return map[int]string{
			1: "a768ef40-b302-11ea",
			2: "ed312110-b310-11ea",
		}
	}

	return vJoyGUIDs
}

// AddBindings adds Gremlins bindings to an ExportDevice
// fullProfile should contain all bindings including keyboard bindings from the module profile
func addBindings(exportDevice *common.ExportDevice, fullProfile *common.Profile, config *common.Config) {
	// Get Gremlins profile path based on sim type and module
	var profilePath string
	if exportDevice.Profile.SimType == common.DCSWorld && exportDevice.Profile.Module != "" {
		profilePath = GetProfilePath(config, common.DCSWorld, exportDevice.Profile.Module)
	} else {
		profilePath = GetProfilePath(config, exportDevice.Profile.SimType, "")
	}

	if profilePath == "" {
		return
	}

	// Build vJoy GUID map from configuration
	vJoyGUIDs := buildVJoyGUIDMap(config)

	// Load Gremlins bindings for ALL devices in the merged profile
	// This handles cases where the representative device is a vJoy but the physical device has Gremlins bindings
	allBindings := make([]*Binding, 0)

	// Collect all device GUIDs from the profile
	deviceGUIDs := make([]string, 0)
	for guid := range exportDevice.Profile.Devices {
		deviceGUIDs = append(deviceGUIDs, guid)
	}

	// If no devices in profile, try the export device GUID
	if len(deviceGUIDs) == 0 {
		deviceGUIDs = append(deviceGUIDs, exportDevice.Device.GUID)
	}

	// Load Gremlins bindings for each device
	for _, deviceGUID := range deviceGUIDs {
		bindings := LoadBindings(profilePath, deviceGUID)
		if len(bindings) > 0 {
			allBindings = append(allBindings, bindings...)
		}
	}

	if len(allBindings) == 0 {
		return
	}

	// Build a map of mode switchers: mode name -> button that activates it
	// Load ALL mode switchers from the entire Gremlins profile, not just from devices in this export
	// This ensures that bindings using modes activated by other devices still get colored correctly
	modeSwitchers := loadAllModeSwitchers(profilePath)

	// Convert Gremlins bindings to regular bindings
	for _, gb := range allBindings {
		// Handle mode switchers - display them as colored modifier bindings
		// (equivalent to DCS "Modifier" buttons)
		if gb.IsModeSwitcher {
			modifierKey := fmt.Sprintf("GREMLINS_MODE_%s", gb.SwitchesTo)

			// Create a binding with "Modifier" prefix so export.go renders it with color
			binding := common.Binding{
				DeviceGUID:  gb.DeviceGUID,
				DeviceName:  gb.DeviceName,
				InputType:   gb.InputType,
				InputID:     gb.InputID,
				Action:      fmt.Sprintf("Modifier %s", modifierKey), // Format recognized by export.go
				Description: fmt.Sprintf("Gremlins Mode: %s", gb.SwitchesTo),
				ModifierKey: modifierKey, // Mark this as a modifier definition
			}

			// Register in ModifierDeviceMap for legend display
			if exportDevice.Profile.ModifierDeviceMap == nil {
				exportDevice.Profile.ModifierDeviceMap = make(map[string]common.ModifierInfo)
			}
			exportDevice.Profile.ModifierDeviceMap[modifierKey] = common.ModifierInfo{
				DeviceGUID: gb.DeviceGUID,
				DeviceName: gb.DeviceName,
				Key:        fmt.Sprintf("BTN%s", gb.InputID),
				IsSwitch:   true,
			}

			exportDevice.Profile.Bindings = append(exportDevice.Profile.Bindings, binding)
			continue
		}

		binding := common.Binding{
			DeviceGUID:  gb.DeviceGUID,
			DeviceName:  gb.DeviceName,
			InputType:   gb.InputType,
			InputID:     gb.InputID,
			Action:      "Gremlins",
			Description: "",
		}

		// If binding is in a non-Base mode, create a modifier
		if gb.Mode != "" && gb.Mode != "Base" {
			if switcher, found := modeSwitchers[gb.Mode]; found {
				// Create modifier key in format similar to DCS: GREMLINS_MODE_Shift
				modifierKey := fmt.Sprintf("GREMLINS_MODE_%s", gb.Mode)
				binding.Modifiers = []common.Modifier{
					{
						Keys:     []string{modifierKey},
						Action:   fmt.Sprintf("Gremlins Mode: %s", gb.Mode),
						IsSwitch: true, // Modes act like switches in Gremlins
					},
				}

				// Store mode switcher info in ModifierDeviceMap if not already present
				if exportDevice.Profile.ModifierDeviceMap == nil {
					exportDevice.Profile.ModifierDeviceMap = make(map[string]common.ModifierInfo)
				}
				if _, exists := exportDevice.Profile.ModifierDeviceMap[modifierKey]; !exists {
					exportDevice.Profile.ModifierDeviceMap[modifierKey] = common.ModifierInfo{
						DeviceGUID: switcher.DeviceGUID,
						DeviceName: switcher.DeviceName,
						Key:        fmt.Sprintf("BTN%s", switcher.InputID),
						IsSwitch:   true,
					}
				}
			}
		}

		// Set virtual device information for CSV export
		switch {
		case gb.VJoyDevice > 0:
			// Use GUID prefix for consistent naming with IL-2 bindings
			if vjoyGUID, ok := vJoyGUIDs[gb.VJoyDevice]; ok && len(vjoyGUID) >= 8 {
				// Extract first 8 chars of GUID
				guidPrefix := strings.ToLower(vjoyGUID[:8])
				binding.VirtualDevice = fmt.Sprintf("vJoy Device %s", guidPrefix)
			} else {
				binding.VirtualDevice = fmt.Sprintf("vJoy Device #%d", gb.VJoyDevice)
			}
			switch {
			case gb.VJoyButton > 0:
				binding.VirtualInput = fmt.Sprintf("BTN%d", gb.VJoyButton)
			case gb.VJoyAxis > 0:
				binding.VirtualInput = fmt.Sprintf("Axis%d", gb.VJoyAxis)
			case gb.VJoyHat > 0:
				// Include direction in VirtualInput (e.g., "POV_1_U")
				// gb.InputID is in format "hatNum_direction" (e.g., "1_U")
				binding.VirtualInput = fmt.Sprintf("POV_%s", gb.InputID)
			}
		case gb.KeyboardKey != "":
			binding.VirtualDevice = "Keyboard"
			binding.VirtualInput = gb.KeyboardKey
		case gb.MouseButton != "":
			binding.VirtualDevice = "Mouse"
			binding.VirtualInput = gb.MouseButton
		}

		// Build description based on binding type
		var actionDesc string

		// vJoy remap - find the actual action in the simulator
		switch {
		case gb.VJoyDevice > 0:
			switch {
			case gb.VJoyButton > 0:
				// Search in both the full profile AND the export device profile
				actionDesc = findVJoyActionInSimulator(gb.VJoyDevice, gb.VJoyButton, fullProfile.Bindings, vJoyGUIDs)
				if actionDesc == "" {
					// Try searching in the current export device's profile bindings
					actionDesc = findVJoyActionInSimulator(gb.VJoyDevice, gb.VJoyButton, exportDevice.Profile.Bindings, vJoyGUIDs)
				}
			case gb.VJoyAxis > 0:
				actionDesc = findVJoyAxisActionInSimulator(gb.VJoyDevice, gb.VJoyAxis, fullProfile.Bindings, vJoyGUIDs)
				if actionDesc == "" {
					actionDesc = findVJoyAxisActionInSimulator(gb.VJoyDevice, gb.VJoyAxis, exportDevice.Profile.Bindings, vJoyGUIDs)
				}
			case gb.VJoyHat > 0:
				// gb.InputID is in format "hatNum_direction" (e.g., "1_U")
				// Extract direction from InputID
				direction := ""
				if parts := strings.Split(gb.InputID, "_"); len(parts) >= 2 {
					direction = parts[1]
				}
				actionDesc = findVJoyHatActionInSimulator(gb.VJoyDevice, gb.VJoyHat, direction, fullProfile.Bindings, vJoyGUIDs)
				if actionDesc == "" {
					actionDesc = findVJoyHatActionInSimulator(gb.VJoyDevice, gb.VJoyHat, direction, exportDevice.Profile.Bindings, vJoyGUIDs)
				}
			}

			// If no action found, show vJoy mapping as fallback with "(unassigned)" prefix
			if actionDesc == "" {
				switch {
				case gb.VJoyButton > 0:
					actionDesc = fmt.Sprintf("(unassigned) vJoy%d BTN%d", gb.VJoyDevice, gb.VJoyButton)
				case gb.VJoyAxis > 0:
					actionDesc = fmt.Sprintf("(unassigned) vJoy%d Axis%d", gb.VJoyDevice, gb.VJoyAxis)
				case gb.VJoyHat > 0:
					actionDesc = fmt.Sprintf("(unassigned) vJoy%d POV%d", gb.VJoyDevice, gb.VJoyHat)
				}
			}
		case gb.KeyboardKey != "":
			// Keyboard mapping - try to find matching action
			actionDesc = findSimulatorActionForGremlins(gb, fullProfile.Bindings)
			if actionDesc == "" {
				actionDesc = gb.KeyboardKey
			}
		case gb.MouseButton != "":
			actionDesc = gb.MouseButton
		}

		// Build final description
		// When simulator actions are found, use them directly without Gremlins description
		// Gremlins description is only used as fallback when no simulator action is found
		switch {
		case actionDesc != "":
			binding.Description = actionDesc
		case gb.Description != "":
			binding.Description = gb.Description
		default:
			binding.Description = "Gremlins"
		}

		exportDevice.Profile.Bindings = append(exportDevice.Profile.Bindings, binding)
	}
}

// findSimulatorActionForGremlins tries to find a simulator action that matches the Gremlins keyboard/mouse binding
func findSimulatorActionForGremlins(gb *Binding, simBindings []common.Binding) string {
	if gb.KeyboardKey == "" {
		// No keyboard key, can't match (mouse bindings are harder to match)
		return ""
	}

	// Build a map of keyboard bindings from simulator
	// For IL-2: key_lshift+key_quote -> "Action Description"
	// For DCS: KEY_Escape -> "Action Description"
	keyboardBindings := make(map[string]string)

	for _, simBinding := range simBindings {
		// Look for keyboard bindings (DeviceGUID == "keyboard")
		if simBinding.DeviceGUID == "keyboard" && simBinding.InputID != "" {
			keyName := simBinding.InputID

			// Store original IL-2 format (key_lshift+key_quote)
			keyboardBindings[keyName] = simBinding.Description

			// Also store DCS format if it has KEY_ prefix
			if after, ok := strings.CutPrefix(keyName, "KEY_"); ok {
				normalizedKey := common.NormalizeKeyNameForMatching(after)
				keyboardBindings[normalizedKey] = simBinding.Description
			}
		}
	}

	// Try to match the Gremlins key combination
	// Convert "LShift + Quote" to IL-2 format: "key_lshift+key_quote"
	gremlinsKeyIL2 := convertGremlinsToIL2Format(gb.KeyboardKey)
	if description, found := keyboardBindings[gremlinsKeyIL2]; found {
		return description
	}

	// Try DCS/simple format
	gremlinsKeyNormalized := common.NormalizeKeyNameForMatching(gb.KeyboardKey)
	if description, found := keyboardBindings[gremlinsKeyNormalized]; found {
		return description
	}

	// Try case-insensitive match
	for key, description := range keyboardBindings {
		if strings.EqualFold(key, gremlinsKeyNormalized) {
			return description
		}
	}

	return ""
}

// findVJoyActionInSimulator finds all simulator actions mapped to a vJoy button
// Returns all matching actions joined with newline for multi-line display
func findVJoyActionInSimulator(vJoyDevice int, vJoyButton int, simBindings []common.Binding, vJoyGUIDs map[int]string) string {
	targetGUID := vJoyGUIDs[vJoyDevice]
	if targetGUID == "" {
		return ""
	}

	// Collect all matching actions (there can be multiple actions on the same vJoy button)
	actions := []string{}
	seen := make(map[string]bool) // Avoid duplicates

	// Look for bindings with matching vJoy device and button
	for _, simBinding := range simBindings {
		deviceGUID := strings.ToLower(simBinding.DeviceGUID)
		deviceName := strings.ToLower(simBinding.DeviceName)

		// Check if this is the right vJoy device by GUID
		if strings.Contains(deviceGUID, targetGUID) ||
			(strings.Contains(deviceName, "vjoy") && strings.Contains(deviceGUID, "vjoy")) {
			// Match button number
			if simBinding.InputType == common.Button && simBinding.InputID == fmt.Sprintf("%d", vJoyButton) {
				if !seen[simBinding.Description] {
					seen[simBinding.Description] = true
					actions = append(actions, simBinding.Description)
				}
			}
		}
	}

	return strings.Join(actions, "\n")
}

// findVJoyAxisActionInSimulator finds the simulator action mapped to a vJoy axis
func findVJoyAxisActionInSimulator(vJoyDevice int, vJoyAxis int, simBindings []common.Binding, vJoyGUIDs map[int]string) string {
	targetGUID := vJoyGUIDs[vJoyDevice]
	if targetGUID == "" {
		return ""
	}

	// Look for bindings with matching vJoy device and axis
	for _, simBinding := range simBindings {
		deviceGUID := strings.ToLower(simBinding.DeviceGUID)
		deviceName := strings.ToLower(simBinding.DeviceName)

		// Check if this is the right vJoy device
		if strings.Contains(deviceGUID, targetGUID) ||
			(strings.Contains(deviceName, "vjoy") && strings.Contains(deviceGUID, "vjoy")) {
			if simBinding.InputType == common.Axis {
				// IL-2 uses axis letters (X, Y, Z, RX, RY, RZ, etc.)
				axisID := strings.ToUpper(simBinding.InputID)

				// Map vJoy axis numbers to letters (common mapping)
				axisNumberToLetter := map[int]string{
					1: "X",
					2: "Y",
					3: "Z",
					4: "RX",
					5: "RY",
					6: "RZ",
					7: "SLIDER_1",
					8: "SLIDER_2",
				}

				expectedAxisLetter := axisNumberToLetter[vJoyAxis]
				if expectedAxisLetter != "" && axisID == expectedAxisLetter {
					return simBinding.Description
				}
			}
		}
	}

	return ""
}

// findVJoyHatActionInSimulator finds all simulator actions mapped to a vJoy hat direction
// direction is "U", "D", "L", "R" for up, down, left, right
// Returns all matching actions joined with newline for multi-line display
func findVJoyHatActionInSimulator(vJoyDevice int, vJoyHat int, direction string, simBindings []common.Binding, vJoyGUIDs map[int]string) string {
	targetGUID := vJoyGUIDs[vJoyDevice]
	if targetGUID == "" {
		return ""
	}

	// Build expected InputID for matching (e.g., "1_U" for hat 1, direction up)
	expectedInputID := fmt.Sprintf("%d_%s", vJoyHat, direction)

	// Collect all matching actions (there can be multiple actions on the same vJoy hat direction)
	actions := []string{}
	seen := make(map[string]bool) // Avoid duplicates

	// Look for bindings with matching vJoy device and hat direction
	for _, simBinding := range simBindings {
		deviceGUID := strings.ToLower(simBinding.DeviceGUID)
		deviceName := strings.ToLower(simBinding.DeviceName)

		// Check if this is the right vJoy device
		if strings.Contains(deviceGUID, targetGUID) ||
			(strings.Contains(deviceName, "vjoy") && strings.Contains(deviceGUID, "vjoy")) {
			if simBinding.InputType == common.Hat {
				// Match exact InputID (e.g., "1_U")
				if simBinding.InputID == expectedInputID {
					if !seen[simBinding.Description] {
						seen[simBinding.Description] = true
						actions = append(actions, simBinding.Description)
					}
				}
			}
		}
	}

	return strings.Join(actions, "\n")
}

// convertGremlinsToIL2Format converts Gremlins format to IL-2 format
// "LShift + Quote" -> "key_lshift+key_quote"
func convertGremlinsToIL2Format(gremlinsKey string) string {
	// Split by " + " for combinations
	parts := strings.Split(gremlinsKey, " + ")

	il2Parts := make([]string, len(parts))
	for i, part := range parts {
		// Convert to IL-2 key name
		il2Parts[i] = "key_" + strings.ToLower(gremlinsKeyToIL2Key(part))
	}

	return strings.Join(il2Parts, "+")
}

// gremlinsKeyToIL2Key converts Gremlins key names to IL-2 key names
func gremlinsKeyToIL2Key(key string) string {
	// Map special Gremlins names to IL-2 names
	keyMap := map[string]string{
		"LShift":    "lshift",
		"RShift":    "rshift",
		"LCtrl":     "lcontrol",
		"RCtrl":     "rcontrol",
		"LAlt":      "lalt",
		"RAlt":      "ralt",
		"LWin":      "lwin",
		"RWin":      "rwin",
		"LBracket":  "lbracket",
		"RBracket":  "rbracket",
		"Quote":     "quote",
		"Semicolon": "semicolon",
		"Comma":     "comma",
		"Period":    "period",
		"Slash":     "slash",
		"Backslash": "backslash",
		"Grave":     "grave",
		"Minus":     "minus",
		"Equals":    "equals",
		"Space":     "space",
		"Enter":     "return",
		"Backspace": "back",
		"ESC":       "escape",
		"CapsLock":  "capital",
	}

	if il2Key, found := keyMap[key]; found {
		return il2Key
	}

	// For letters and numbers, just lowercase
	return strings.ToLower(key)
}

// LoadBindingsForDevice loads Gremlins bindings for a specific device using config
// Returns empty slice if no Gremlins profile is configured or device has no bindings
func LoadBindingsForDevice(deviceGUID string, config *common.Config) []*Binding {
	if config == nil {
		return nil
	}

	// Check all simulators and modules for Gremlins profiles
	var allBindings []*Binding

	// Check IL-2 (Great Battles and Korea)
	for _, simType := range []common.SimulationType{common.IL2Sturmovik, common.IL2Korea} {
		il2Profile := GetProfilePath(config, simType, "")
		if il2Profile != "" {
			bindings := LoadBindings(il2Profile, deviceGUID)
			allBindings = append(allBindings, bindings...)
		}
	}

	// Check DCS modules
	if config.Simulators != nil {
		if dcsConfig := config.Simulators["dcs_world"]; dcsConfig != nil && dcsConfig.Modules != nil {
			for moduleName := range dcsConfig.Modules {
				moduleProfile := GetProfilePath(config, common.DCSWorld, moduleName)
				if moduleProfile != "" {
					bindings := LoadBindings(moduleProfile, deviceGUID)
					allBindings = append(allBindings, bindings...)
				}
			}
		}
	}

	return allBindings
}

// loadAllModeSwitchers loads ALL mode switchers from the entire Gremlins profile
// This is used to ensure color consistency even when the mode activator is on a different device
func loadAllModeSwitchers(profilePath string) map[string]*Binding {
	modeSwitchers := make(map[string]*Binding)

	if profilePath == "" {
		return modeSwitchers
	}

	// Parse the entire Gremlins profile to get all bindings
	allBindings, err := ParseProfile(profilePath)
	if err != nil {
		return modeSwitchers
	}

	// Extract all mode switchers
	for _, gb := range allBindings {
		if gb.IsModeSwitcher {
			modeSwitchers[gb.SwitchesTo] = gb
		}
	}

	return modeSwitchers
}
