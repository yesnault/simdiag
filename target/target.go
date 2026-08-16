package target

import (
	"fmt"
	"simdiag/common"
	"strings"
)

// addBindings adds TARGET bindings to an ExportDevice
// fullProfile should contain all bindings including keyboard bindings from the module profile
func addBindings(exportDevice *common.ExportDevice, fullProfile *common.Profile, config *common.Config) {
	profilePath, deviceNumberToGUID := getProfilePathAndMappings(exportDevice, config)
	if profilePath == "" {
		return
	}

	targetDeviceNumbers := findMatchingTargetDevices(exportDevice, deviceNumberToGUID)
	if len(targetDeviceNumbers) == 0 {
		return
	}

	allBindings := loadAllBindings(profilePath, targetDeviceNumbers)
	if len(allBindings) == 0 {
		return
	}

	layerSwitchers := ParseLayerSwitchers(profilePath)
	hasSwitchI := convertAndAddBindings(exportDevice, fullProfile, allBindings, deviceNumberToGUID)

	if hasSwitchI {
		addSwitchIBinding(exportDevice, layerSwitchers, deviceNumberToGUID)
	}

	// Transfer bindings from TARGET Combined device to physical joystick
	transferCombinedBindings(exportDevice, fullProfile)
}

// getProfilePathAndMappings gets TARGET profile path and device mappings
func getProfilePathAndMappings(exportDevice *common.ExportDevice, config *common.Config) (string, map[int]string) {
	profilePath := GetProfilePath(config, exportDevice.Profile.SimType)
	deviceMappings := GetTargetDeviceMappings(config)

	deviceNumberToGUID := make(map[int]string)
	for _, mapping := range deviceMappings {
		deviceNumberToGUID[mapping.DeviceNumber] = mapping.DeviceGUID
	}

	return profilePath, deviceNumberToGUID
}

// findMatchingTargetDevices finds TARGET device numbers that match profile devices
func findMatchingTargetDevices(exportDevice *common.ExportDevice, deviceNumberToGUID map[int]string) []int {
	deviceGUIDs := collectDeviceGUIDs(exportDevice)
	targetDeviceNumbers := make([]int, 0)

	for _, guid := range deviceGUIDs {
		for deviceNum, mappedGUID := range deviceNumberToGUID {
			if strings.EqualFold(guid, mappedGUID) || common.MatchGUIDPartial(guid, mappedGUID) {
				targetDeviceNumbers = append(targetDeviceNumbers, deviceNum)
				break
			}
		}
	}

	return targetDeviceNumbers
}

// collectDeviceGUIDs collects all device GUIDs from the profile
func collectDeviceGUIDs(exportDevice *common.ExportDevice) []string {
	deviceGUIDs := make([]string, 0)
	for guid := range exportDevice.Profile.Devices {
		deviceGUIDs = append(deviceGUIDs, guid)
	}

	if len(deviceGUIDs) == 0 {
		deviceGUIDs = append(deviceGUIDs, exportDevice.Device.GUID)
	}

	return deviceGUIDs
}

// loadAllBindings loads TARGET bindings for matched devices
func loadAllBindings(profilePath string, targetDeviceNumbers []int) []*Binding {
	allBindings := make([]*Binding, 0)

	for _, deviceNum := range targetDeviceNumbers {
		bindings := LoadBindings(profilePath, deviceNum)
		if len(bindings) > 0 {
			allBindings = append(allBindings, bindings...)
		}
	}

	return allBindings
}

// convertAndAddBindings converts TARGET bindings and adds them to the profile
func convertAndAddBindings(exportDevice *common.ExportDevice, fullProfile *common.Profile, allBindings []*Binding, deviceNumberToGUID map[int]string) bool {
	hasSwitchI := false

	for _, tb := range allBindings {
		bindings := convertBinding(exportDevice, fullProfile, tb, deviceNumberToGUID, &hasSwitchI)
		if len(bindings) > 0 {
			exportDevice.Profile.Bindings = append(exportDevice.Profile.Bindings, bindings...)
		}
	}

	return hasSwitchI
}

// convertBinding converts a single TARGET binding to common bindings
// Returns multiple bindings if the same key combo maps to multiple simulator actions
func convertBinding(exportDevice *common.ExportDevice, fullProfile *common.Profile, tb *Binding, deviceNumberToGUID map[int]string, hasSwitchI *bool) []common.Binding {
	deviceGUID := deviceNumberToGUID[tb.DeviceNumber]
	if deviceGUID == "" {
		return nil
	}

	deviceName := findDeviceName(exportDevice, deviceGUID, tb.DeviceNumber)
	action := determineAction(tb)

	// Get all matching simulator actions
	actionDescs := findSimulatorActionForTarget(tb, fullProfile.Bindings)
	if len(actionDescs) == 0 {
		actionDescs = findSimulatorActionForTarget(tb, exportDevice.Profile.Bindings)
	}

	// If no simulator actions found, create one binding with default description
	if len(actionDescs) == 0 {
		binding := common.Binding{
			DeviceGUID:  deviceGUID,
			DeviceName:  deviceName,
			InputType:   tb.InputType,
			InputID:     tb.InputID,
			Action:      action,
			Description: "",
		}

		addModifiers(&binding, tb, hasSwitchI)
		setVirtualDeviceInfo(&binding, tb)
		setDefaultDescription(&binding, tb)

		return []common.Binding{binding}
	}

	// Create one binding for each simulator action
	var bindings []common.Binding
	for _, desc := range actionDescs {
		binding := common.Binding{
			DeviceGUID:  deviceGUID,
			DeviceName:  deviceName,
			InputType:   tb.InputType,
			InputID:     tb.InputID,
			Action:      action,
			Description: desc,
		}

		addModifiers(&binding, tb, hasSwitchI)
		setVirtualDeviceInfo(&binding, tb)

		// Add (MID) suffix for mid-position switches
		if IsMidPosition(tb.InputName) {
			binding.Description += " (MID)"
		}

		bindings = append(bindings, binding)
	}

	return bindings
}

// findDeviceName finds the device name from GUID or TARGET device number
func findDeviceName(exportDevice *common.ExportDevice, deviceGUID string, deviceNumber int) string {
	if device, exists := exportDevice.Profile.Devices[deviceGUID]; exists {
		return device.Name
	}
	return DeviceNumberToName(deviceNumber)
}

// determineAction determines the action string based on trigger type
func determineAction(tb *Binding) string {
	switch tb.Trigger {
	case "I":
		return "TARGET_I"
	case "O":
		return "TARGET_O"
	default:
		return "TARGET"
	}
}

// addModifiers adds modifiers to the binding if needed
func addModifiers(binding *common.Binding, tb *Binding, hasSwitchI *bool) {
	if tb.LayerInfo.HasSwitchActive() {
		*hasSwitchI = true
		binding.Modifiers = []common.Modifier{
			{
				Keys:     []string{"TARGET_SWITCH_I"},
				Action:   "TARGET Switch (I)",
				IsSwitch: true,
			},
		}
	}
}

// setVirtualDeviceInfo sets virtual device information for CSV export
func setVirtualDeviceInfo(binding *common.Binding, tb *Binding) {
	switch {
	case len(tb.OutputKeys) > 0:
		binding.VirtualDevice = "Keyboard"
		// Normalize key order (modifiers first) for consistent comparison with IL-2/DCS bindings
		normalizedKeys := common.NormalizeKeyOrder(tb.OutputKeys)
		binding.VirtualInput = strings.Join(normalizedKeys, " + ")
	case tb.OutputMouse != "":
		binding.VirtualDevice = "Mouse"
		binding.VirtualInput = tb.OutputMouse
	case tb.OutputJoystick != "":
		binding.VirtualDevice = "Virtual Joystick"
		binding.VirtualInput = tb.OutputJoystick
	}
}

// setDefaultDescription sets a default description when no simulator action is found
func setDefaultDescription(binding *common.Binding, tb *Binding) {
	switch {
	case tb.EventName != "":
		if len(tb.OutputKeys) > 0 {
			// Normalize key order for display
			normalizedKeys := common.NormalizeKeyOrder(tb.OutputKeys)
			binding.Description = fmt.Sprintf("%s (%s)", tb.EventName, strings.Join(normalizedKeys, "+"))
		} else {
			binding.Description = tb.EventName
		}
	case len(tb.OutputKeys) > 0:
		// Normalize key order for display
		normalizedKeys := common.NormalizeKeyOrder(tb.OutputKeys)
		binding.Description = fmt.Sprintf("TARGET: %s", strings.Join(normalizedKeys, "+"))
	case tb.OutputMouse != "":
		binding.Description = fmt.Sprintf("TARGET: %s", tb.OutputMouse)
	default:
		binding.Description = "TARGET"
	}

	// Add (MID) suffix for mid-position switches (3-pos switches mapped to high position)
	if IsMidPosition(tb.InputName) {
		binding.Description += " (MID)"
	}
}

// addSwitchIBinding creates a binding for the I switch button
func addSwitchIBinding(exportDevice *common.ExportDevice, layerSwitchers map[int]LayerSwitcher, deviceNumberToGUID map[int]string) {
	switcher, found := layerSwitchers[2]
	if !found {
		return
	}

	deviceGUID := deviceNumberToGUID[switcher.DeviceNumber]
	if deviceGUID == "" {
		return
	}

	// Only add switch binding to the ExportDevice that owns the switch button
	if !isDeviceInExportDevice(exportDevice, deviceGUID) {
		return
	}

	inputType, inputID := parseButtonID(switcher.ButtonID)
	if inputID == "" {
		return
	}

	deviceName := findDeviceNameForSwitcher(exportDevice, deviceGUID, switcher.DeviceNumber)

	switchBinding := common.Binding{
		DeviceGUID:  deviceGUID,
		DeviceName:  deviceName,
		InputType:   inputType,
		InputID:     inputID,
		Action:      "Switch TARGET_SWITCH_I",
		Description: "TARGET Switch (I)",
		Modifiers:   []common.Modifier{},
		ModifierKey: "TARGET_SWITCH_I",
	}

	exportDevice.Profile.Bindings = append(exportDevice.Profile.Bindings, switchBinding)
	common.Printf("  ℹ TARGET I switch button: %s on %s\n", switcher.ButtonID, deviceName)
}

// parseButtonID parses a button ID string to input type and ID
func parseButtonID(buttonID string) (common.InputType, string) {
	switch {
	case strings.HasPrefix(buttonID, "BTN"):
		return common.Button, strings.TrimPrefix(buttonID, "BTN")
	case strings.HasPrefix(buttonID, "POV_"):
		return common.Hat, strings.TrimPrefix(buttonID, "POV_")
	case strings.HasPrefix(buttonID, "AXIS_"):
		return common.Axis, strings.TrimPrefix(buttonID, "AXIS_")
	default:
		return common.Button, ""
	}
}

// findDeviceNameForSwitcher finds device name for layer switcher
func findDeviceNameForSwitcher(exportDevice *common.ExportDevice, deviceGUID string, deviceNumber int) string {
	for _, device := range exportDevice.Profile.Devices {
		if device.GUID == deviceGUID {
			return device.Name
		}
	}
	return DeviceNumberToName(deviceNumber)
}

// transferCombinedBindings transfers bindings from TARGET Combined device to physical devices
// TARGET creates a "Combined" virtual device that merges Joystick and Throttle
// IL-2 assigns axes to this Combined device, so we need to transfer them to the correct physical devices
// - Axes X and Y (roll/pitch) → Joystick
// - Axes Z, Q, T, etc. (throttle/mixture/prop) → Throttle
func transferCombinedBindings(exportDevice *common.ExportDevice, fullProfile *common.Profile) {
	// Find physical joystick and throttle devices in exportDevice
	var joystickGUID, throttleGUID string
	var joystickDevice, throttleDevice *common.Device

	for guid, device := range exportDevice.Profile.Devices {
		nameLower := strings.ToLower(device.Name)
		if strings.Contains(nameLower, "joystick") && !strings.Contains(nameLower, "virtual") {
			joystickGUID = guid
			joystickDevice = device
		} else if strings.Contains(nameLower, "throttle") && !strings.Contains(nameLower, "virtual") {
			throttleGUID = guid
			throttleDevice = device
		}
	}

	// Joystick axes (roll and pitch)
	joystickAxes := map[string]bool{"X": true, "Y": true}

	// Transfer bindings from Combined device in fullProfile
	for _, binding := range fullProfile.Bindings {
		// Check if this is a Combined device binding
		if common.IsVirtualDevice(binding.DeviceName) && strings.Contains(strings.ToLower(binding.DeviceName), "combined") {
			// This is a binding from the Combined device
			// Only transfer axis bindings (not buttons, as those are on the actual devices)
			if binding.InputType == common.Axis {
				axisID := strings.ToUpper(binding.InputID)

				// Check if this is a joystick axis (X or Y)
				if joystickAxes[axisID] && joystickGUID != "" && joystickDevice != nil {
					transferredBinding := binding
					transferredBinding.DeviceGUID = joystickGUID
					transferredBinding.DeviceName = joystickDevice.Name
					exportDevice.Profile.Bindings = append(exportDevice.Profile.Bindings, transferredBinding)
				} else if !joystickAxes[axisID] && throttleGUID != "" && throttleDevice != nil {
					// All other axes go to throttle
					transferredBinding := binding
					transferredBinding.DeviceGUID = throttleGUID
					transferredBinding.DeviceName = throttleDevice.Name
					exportDevice.Profile.Bindings = append(exportDevice.Profile.Bindings, transferredBinding)
				}
			}
		}
	}
}

// isDeviceInExportDevice checks if a device GUID belongs to the current ExportDevice
func isDeviceInExportDevice(exportDevice *common.ExportDevice, deviceGUID string) bool {
	for guid := range exportDevice.Profile.Devices {
		if strings.EqualFold(guid, deviceGUID) || common.MatchGUIDPartial(guid, deviceGUID) {
			return true
		}
	}
	return strings.EqualFold(exportDevice.Device.GUID, deviceGUID) ||
		common.MatchGUIDPartial(exportDevice.Device.GUID, deviceGUID)
}

// findSimulatorActionForTarget tries to find simulator actions that match the TARGET keyboard binding
// Returns ALL matching actions (since one key combo can map to multiple functions in IL-2)
// TARGET stores keys in the layout its profile was authored in (e.g., ")" on AZERTY)
// IL-2 uses QWERTY key names internally (e.g., "key_equals" for the physical = key)
// So we convert from the profile's layout to QWERTY for matching
func findSimulatorActionForTarget(tb *Binding, simBindings []common.Binding) []string {
	var results []string

	// Check for virtual joystick button output (vJoy BTN98, etc.)
	if tb.OutputJoystick != "" && strings.HasPrefix(tb.OutputJoystick, "vJoy BTN") {
		// Extract button number (e.g., "vJoy BTN98" -> "98")
		buttonNumStr := strings.TrimPrefix(tb.OutputJoystick, "vJoy BTN")

		// Search for joystick button bindings
		// IL-2 parser already converts joy3_b97 (0-based) to InputID=98 (1-based)
		// So TARGET BTN98 matches IL-2 InputID "98"
		for _, simBinding := range simBindings {
			if simBinding.InputType == common.Button && simBinding.DeviceGUID != "keyboard" {
				if simBinding.InputID == buttonNumStr {
					results = append(results, simBinding.Description)
				}
			}
		}
		return results
	}

	if len(tb.OutputKeys) == 0 {
		return results
	}

	// Convert TARGET keys from the profile's layout to QWERTY (IL-2's internal format)
	convertedKeys := ConvertKeysForLayout(tb.OutputKeys, tb.KeyboardLayout, KeyboardQWERTY)

	// Normalize key order (modifiers first) for consistent matching
	convertedKeys = common.NormalizeKeyOrder(convertedKeys)

	// Build the keyboard key combination (with converted and normalized keys)
	targetKey := strings.Join(convertedKeys, " + ")

	// Build a map of keyboard bindings from simulator (key -> list of descriptions)
	keyboardBindings := make(map[string][]string)

	for _, simBinding := range simBindings {
		// Look for keyboard bindings (DeviceGUID == "keyboard")
		if simBinding.DeviceGUID == "keyboard" && simBinding.InputID != "" {
			keyName := simBinding.InputID

			// Store original format
			keyboardBindings[keyName] = append(keyboardBindings[keyName], simBinding.Description)

			// Store normalized format for matching
			normalizedKey := common.NormalizeKeyNameForMatching(keyName)
			keyboardBindings[normalizedKey] = append(keyboardBindings[normalizedKey], simBinding.Description)

			// Also store IL-2 format if applicable
			if strings.HasPrefix(keyName, "key_") {
				// Convert IL-2 format to standard format
				il2Key := strings.TrimPrefix(keyName, "key_")
				standardKey := il2KeyToStandard(il2Key)
				keyboardBindings[standardKey] = append(keyboardBindings[standardKey], simBinding.Description)
			}
		}
	}

	// Try to match the TARGET key combination
	// First try direct match
	if descriptions, found := keyboardBindings[targetKey]; found {
		results = append(results, descriptions...)
		return results
	}

	// Convert TARGET keys to standard format for comparison (using converted keys)
	targetKeyNormalized := normalizeTargetKeys(convertedKeys)
	if descriptions, found := keyboardBindings[targetKeyNormalized]; found {
		results = append(results, descriptions...)
		return results
	}

	// Try IL-2 format (using converted keys)
	targetKeyIL2 := convertTargetToIL2Format(convertedKeys)
	if descriptions, found := keyboardBindings[targetKeyIL2]; found {
		results = append(results, descriptions...)
		return results
	}

	// Try case-insensitive match
	for key, descriptions := range keyboardBindings {
		if strings.EqualFold(key, targetKeyNormalized) {
			results = append(results, descriptions...)
			return results
		}
	}

	return results
}

// normalizeTargetKeys normalizes TARGET key names to standard format
func normalizeTargetKeys(keys []string) string {
	normalized := make([]string, len(keys))
	for i, key := range keys {
		normalized[i] = common.NormalizeKeyNameForMatching(key)
	}
	return strings.Join(normalized, " + ")
}

// convertTargetToIL2Format converts TARGET format to IL-2 format
// ["LAlt", "Space"] -> "key_lalt+key_space"
func convertTargetToIL2Format(keys []string) string {
	il2Parts := make([]string, len(keys))
	for i, key := range keys {
		il2Parts[i] = "key_" + strings.ToLower(targetKeyToIL2Key(key))
	}
	return strings.Join(il2Parts, "+")
}

// targetKeyToIL2Key converts TARGET key names to IL-2 key names
func targetKeyToIL2Key(key string) string {
	if il2Key, found := common.IL2KeyName(key); found {
		return il2Key
	}

	// The keypad and the punctuation are TARGET's own vocabulary; the modifiers
	// and named keys above are shared with the Gremlins enricher.
	keyMap := map[string]string{
		"Num0":     "numpad0",
		"Num1":     "numpad1",
		"Num2":     "numpad2",
		"Num3":     "numpad3",
		"Num4":     "numpad4",
		"Num5":     "numpad5",
		"Num6":     "numpad6",
		"Num7":     "numpad7",
		"Num8":     "numpad8",
		"Num9":     "numpad9",
		"Num.":     "decimal",
		"NumEnter": "numpadenter",
		"Num+":     "add",
		"Num-":     "subtract",
		"Num*":     "multiply",
		"Num/":     "divide",
		// Special characters
		"=":  "equals",
		"-":  "minus",
		")":  "equals", // AZERTY ")" is same physical key as QWERTY "="
		"(":  "minus",  // AZERTY "(" is same physical key as QWERTY "-"
		"[":  "lbracket",
		"]":  "rbracket",
		";":  "semicolon",
		"'":  "apostrophe",
		",":  "comma",
		".":  "period",
		"/":  "slash",
		"\\": "backslash",
		"`":  "grave",
	}

	if il2Key, found := keyMap[key]; found {
		return il2Key
	}

	return strings.ToLower(key)
}

// il2KeyToStandard converts IL-2 key names to standard format
func il2KeyToStandard(il2Key string) string {
	return common.StandardKeyName(il2Key)
}

// GetProfilePath returns the TARGET profile path configured for a simulator.
//
// It takes no module: a TARGET script is written for a physical HOTAS, not for an
// aircraft, so it applies to every module of the simulator it is configured on.
func GetProfilePath(config *common.Config, simType common.SimulationType) string {
	return config.TargetProfilePath(simType)
}

// GetTargetDeviceMappings returns the TARGET device mappings for the current context
// Built from DeviceMappings entries that have a DeviceTargetNumber set
func GetTargetDeviceMappings(config *common.Config) []common.TargetDeviceMapping {
	if config == nil {
		return nil
	}

	// Get device mappings from global config
	deviceMappings := config.DeviceMappings

	// Build TargetDeviceMapping list from DeviceMappings with DeviceTargetNumber
	var result []common.TargetDeviceMapping
	for _, dm := range deviceMappings {
		if dm.DeviceTargetNumber != 0 {
			result = append(result, common.TargetDeviceMapping{
				DeviceNumber: dm.DeviceTargetNumber,
				DeviceGUID:   dm.DeviceGUID,
				DeviceName:   dm.DeviceName,
			})
		}
	}

	return result
}
