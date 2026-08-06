package dcs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"simdiag/common"
	"strings"
)

// Patterns used to pull bindings out of DCS .lua files. They are compiled once:
// several of them are applied per binding, inside nested loops.
var (
	// Device / modifier declarations
	deviceGUIDPattern = regexp.MustCompile(`(?i)\{[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}\}`)
	modifierPattern   = regexp.MustCompile(`(?s)\["([^"]+)"\]\s*=\s*\{[^}]*\["device"\]\s*=\s*"([^"]*\{([^}]+)\})"[^}]*\["key"\]\s*=\s*"([^"]+)"[^}]*\["switch"\]\s*=\s*(true|false)[^}]*\}`)

	// Section delimiters
	axisDiffsStartPattern = regexp.MustCompile(`\["axisDiffs"\]\s*=\s*\{`)
	axisBlockStartPattern = regexp.MustCompile(`\["a\d+[^"]*"\]\s*=\s*\{`)
	changedStartPattern   = regexp.MustCompile(`\["changed"\]\s*=\s*\{`)
	addedStartPattern     = regexp.MustCompile(`\["added"\]\s*=\s*\{`)
	keyDiffsPattern       = regexp.MustCompile(`(?s)\["keyDiffs"\]\s*=\s*\{(.*)\},?\s*\}\s*return diff`)
	keyDiffsLoosePattern  = regexp.MustCompile(`(?s)\["keyDiffs"\]\s*=\s*\{(.*)\}`)
	addedBlockPattern     = regexp.MustCompile(`(?s)\["added"\]\s*=\s*\{((?:[^{}]|\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*\})*)\}`)

	// Binding blocks
	diffBlockPattern    = regexp.MustCompile(`(?s)\["d[^"]+"\]\s*=\s*\{(?:[^{}]|\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*\})*\}`)
	anyBlockPattern     = regexp.MustCompile(`(?s)\["[^"]+"\]\s*=\s*\{(?:[^{}]|\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*\})*\}`)
	axisEntryPattern    = regexp.MustCompile(`(?s)\[\d+\]\s*=\s*\{([^{}]*(?:\{[^{}]*(?:\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}[^{}]*)*\}[^{}]*)*)\}`)
	keyEntryPattern     = regexp.MustCompile(`(?s)\[\d+\]\s*=\s*\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}`)
	reformersPattern    = regexp.MustCompile(`(?s)\["reformers"\]\s*=\s*\{(.*?)\}`)
	numberedKeyPattern  = regexp.MustCompile(`\[\d+\]\s*=\s*"([^"]+)"`)
	defaultCombosPatern = regexp.MustCompile(`\{combos\s*=\s*\{\{key\s*=\s*'([^']+)'\}\},\s*down\s*=\s*[^,]+,\s*name\s*=\s*_\('([^']+)'\)`)

	// Field extraction
	namePattern     = regexp.MustCompile(`\["name"\]\s*=\s*"([^"]+)"`)
	axisNamePattern = regexp.MustCompile(`\["name"\]\s*=\s*"([^"]*)"`)
	joyKeyPattern   = regexp.MustCompile(`\["key"\]\s*=\s*"(JOY_[^"]+)"`)
	anyKeyPattern   = regexp.MustCompile(`\["key"\]\s*=\s*"([^"]+)"`) // Accept any key format, not just KEY_

	// Input naming
	sliderPattern       = regexp.MustCompile(`^SLIDER(\d+)$`)
	standardAxisPattern = regexp.MustCompile(`^R?[XYZ]$`)
)

// Parser implements the SimulatorParser interface for DCS World
type Parser struct{}

// NewParser creates a new DCS parser instance
func NewParser() *Parser {
	return &Parser{}
}

// Parse implements SimulatorParser.Parse
func (p *Parser) Parse(basePath string) (*common.ProfileCollection, error) {
	return parseDCS(basePath)
}

// GetName implements SimulatorParser.GetName
func (p *Parser) GetName() string {
	return "DCS World"
}

// parseDCS parses DCS World configuration files
func parseDCS(basePath string) (*common.ProfileCollection, error) {
	collection := &common.ProfileCollection{
		Profiles: make([]*common.Profile, 0),
	}

	// Path to Config/Input
	inputPath := filepath.Join(basePath, "Config", "Input")

	// Check that the folder exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no Config/Input folder in: %s", basePath)
	}

	// List profiles (subfolders)
	entries, err := os.ReadDir(inputPath)
	if err != nil {
		return nil, fmt.Errorf("error reading Input folder: %w", err)
	}

	// Parse each profile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Ignore _easy profiles
		if strings.Contains(entry.Name(), "_easy") {
			continue
		}

		profile, err := parseDCSProfile(filepath.Join(inputPath, entry.Name()), entry.Name())
		if err != nil {
			fmt.Printf("Warning: unable to parse profile %s: %v\n", entry.Name(), err)
			continue
		}

		if profile != nil {
			collection.Profiles = append(collection.Profiles, profile)
		}
	}

	if len(collection.Profiles) == 0 {
		return nil, fmt.Errorf("no valid profile found")
	}

	// AFTER all profiles are parsed, merge ModifierDeviceMaps and create pure modifier/switch bindings
	// This ensures that modifiers defined in one profile can be used in another profile
	mergeModifierDeviceMapsAndCreateBindings(collection.Profiles)

	return collection, nil
}

// parseDCSProfile parses an individual DCS profile
func parseDCSProfile(profilePath, profileName string) (*common.Profile, error) {
	// Path to joystick folder
	joystickPath := filepath.Join(profilePath, "joystick")

	if _, err := os.Stat(joystickPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no joystick folder in profile")
	}

	// Extract module name (skip Default, UiLayer, CommandMenu)
	moduleName := ""
	ignoredProfiles := map[string]bool{
		"Default":     true,
		"UiLayer":     true,
		"CommandMenu": true,
	}

	if !ignoredProfiles[profileName] {
		moduleName = profileName
	}

	profile := &common.Profile{
		Name:     profileName,
		SimType:  common.DCSWorld,
		Module:   moduleName,
		Devices:  make(map[string]*common.Device),
		Bindings: make([]common.Binding, 0),
	}

	// List .lua files in joystick folder
	entries, err := os.ReadDir(joystickPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".lua") {
			continue
		}

		// Parse lua file
		filePath := filepath.Join(joystickPath, entry.Name())
		err := parseDCSLuaFile(filePath, entry.Name(), profile)
		if err != nil {
			fmt.Printf("  Warning: error parsing %s: %v\n", entry.Name(), err)
		}
	}

	// Also parse keyboard folder if it exists
	keyboardPath := filepath.Join(profilePath, "keyboard")
	if _, err := os.Stat(keyboardPath); err == nil {
		keyboardEntries, err := os.ReadDir(keyboardPath)
		if err == nil {
			for _, entry := range keyboardEntries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".lua") {
					continue
				}

				// Read keyboard file and parse keyboard bindings
				keyboardFilePath := filepath.Join(keyboardPath, entry.Name())
				keyboardContent, err := os.ReadFile(keyboardFilePath)
				if err == nil {
					parseKeyboardBindingsFromLua(string(keyboardContent), profile)
				}
			}
		}
	}

	// Also parse default keyboard bindings from DCS installation
	// These provide the base F1-F12 view commands and other defaults
	parseDefaultKeyboardBindings(profile)

	// Parse modifiers.lua to get device ownership of modifier/switch buttons
	modifiersFilePath := filepath.Join(profilePath, "modifiers.lua")
	parseModifiersFile(modifiersFilePath, profile)

	// Note: createPureModifierBindings and createPureSwitchBindings are now called
	// AFTER all profiles are parsed (in mergeModifierDeviceMapsAndCreateBindings)
	// to ensure modifiers defined in one profile can be used in another profile

	return profile, nil
}

// parseModifiersFile parses the modifiers.lua file to extract device ownership of modifiers/switches
func parseModifiersFile(filePath string, profile *common.Profile) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		// File might not exist, that's okay
		return
	}

	fileContent := string(content)

	// modifierPattern matches each modifier entry:
	// ["Touche-JOY_BTN102"] = {
	//     ["device"] = "WINWING ... {530648C0-98C7-11f0-8002-444553540000}",
	//     ["key"] = "JOY_BTN102",
	//     ["switch"] = true,
	// },
	matches := modifierPattern.FindAllStringSubmatch(fileContent, -1)

	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		// modifierName := match[1] // e.g., "Touche-JOY_BTN102" (not used)
		deviceFull := match[2] // e.g., "WINWING Orion Throttle Base II + F15EX HANDLE L + F15EX HANDLE R {530648C0-98C7-11f0-8002-444553540000}"
		guid := match[3]       // e.g., "530648C0-98C7-11f0-8002-444553540000"
		key := match[4]        // e.g., "JOY_BTN102"
		isSwitch := match[5] == "true"

		// Extract device name by removing the GUID part
		deviceName := strings.TrimSpace(strings.Replace(deviceFull, "{"+guid+"}", "", 1))

		// Store this modifier info in the profile for later use
		// We'll add it to a map in the profile structure
		if profile.ModifierDeviceMap == nil {
			profile.ModifierDeviceMap = make(map[string]common.ModifierInfo)
		}

		profile.ModifierDeviceMap[key] = common.ModifierInfo{
			DeviceGUID: guid,
			DeviceName: deviceName,
			Key:        key,
			IsSwitch:   isSwitch,
		}
	}
}

// extractBalancedBraces extracts content between balanced braces starting from a given position
// Returns the content inside the braces and the position after the closing brace
func extractBalancedBraces(content string, startPos int) (string, int) {
	if startPos >= len(content) || content[startPos] != '{' {
		return "", -1
	}

	braceCount := 0
	startContentPos := startPos + 1

	for i := startPos; i < len(content); i++ {
		if content[i] == '{' {
			braceCount++
		} else if content[i] == '}' {
			braceCount--
			if braceCount == 0 {
				// Found matching closing brace
				return content[startContentPos:i], i + 1
			}
		}
	}

	return "", -1 // Unbalanced braces
}

// extractSectionAfter locates pattern in content and returns the balanced-brace
// block that follows it. Returns "" when the section is absent or unbalanced.
func extractSectionAfter(content string, pattern *regexp.Regexp) string {
	match := pattern.FindStringIndex(content)
	if match == nil {
		return ""
	}

	bracePos := strings.Index(content[match[0]:], "{")
	if bracePos < 0 {
		return ""
	}

	block, endPos := extractBalancedBraces(content, match[0]+bracePos)
	if endPos < 0 {
		return ""
	}
	return block
}

// newJoyBinding builds a binding for a DCS JOY_ key. Bindings on the release edge
// carry an "_OFF" input ID and get their action suffixed so both edges stay
// distinguishable on the diagram.
func newJoyBinding(guid, deviceName, joyKey, actionName string) common.Binding {
	inputType, inputID := parseDCSJoyInput(joyKey)

	displayAction := actionName
	if strings.HasSuffix(inputID, "_OFF") {
		displayAction = actionName + " (OFF)"
	}

	return common.Binding{
		DeviceGUID: guid,
		DeviceName: deviceName,
		InputType:  inputType,
		InputID:    inputID,
		Action:     displayAction,
	}
}

// extractAxisBlocks splits the axisDiffs section into one self-contained block per
// axis: ["a2001cdnil"] = { ... }.
func extractAxisBlocks(axisDiffsContent string) []string {
	var axisBlocks []string

	for _, match := range axisBlockStartPattern.FindAllStringIndex(axisDiffsContent, -1) {
		bracePos := strings.Index(axisDiffsContent[match[0]:], "{")
		if bracePos < 0 {
			continue
		}

		actualBracePos := match[0] + bracePos
		blockContent, endPos := extractBalancedBraces(axisDiffsContent, actualBracePos)
		if endPos < 0 {
			continue
		}

		// Rebuild the full block: key name + "{" + content + "}"
		keyPart := axisDiffsContent[match[0]:actualBracePos]
		axisBlocks = append(axisBlocks, keyPart+"{"+blockContent+"}")
	}

	return axisBlocks
}

// parseAxisDiffs extracts axis bindings from axisDiffs section
func parseAxisDiffs(fileContent, guid, deviceName string, profile *common.Profile) {
	axisDiffsContent := extractSectionAfter(fileContent, axisDiffsStartPattern)
	if axisDiffsContent == "" {
		return
	}

	for _, axisBlock := range extractAxisBlocks(axisDiffsContent) {
		nameMatch := axisNamePattern.FindStringSubmatch(axisBlock)
		if len(nameMatch) < 2 {
			continue
		}
		actionName := nameMatch[1]

		// Active bindings live under "changed" or, failing that, "added".
		// Anything under "removed" is ignored.
		bindingContent := extractSectionAfter(axisBlock, changedStartPattern)
		if bindingContent == "" {
			bindingContent = extractSectionAfter(axisBlock, addedStartPattern)
		}
		if bindingContent == "" {
			continue
		}

		// Numbered entries [1] = { ... }, [2] = { ... }; older files carry the key
		// directly in the section.
		entryMatches := axisEntryPattern.FindAllStringSubmatch(bindingContent, -1)
		if len(entryMatches) == 0 {
			if keyMatch := joyKeyPattern.FindStringSubmatch(bindingContent); len(keyMatch) >= 2 {
				profile.Bindings = append(profile.Bindings, newJoyBinding(guid, deviceName, keyMatch[1], actionName))
			}
			continue
		}

		for _, entryMatch := range entryMatches {
			if len(entryMatch) < 2 {
				continue
			}

			keyMatch := joyKeyPattern.FindStringSubmatch(entryMatch[1])
			if len(keyMatch) < 2 {
				continue
			}

			profile.Bindings = append(profile.Bindings, newJoyBinding(guid, deviceName, keyMatch[1], actionName))
		}
	}
}

// parseKeyDiffs extracts button bindings from keyDiffs section
func parseKeyDiffs(fileContent, guid, deviceName string, profile *common.Profile) []common.Binding {
	// First, extract the ["keyDiffs"] block
	keyDiffsMatch := keyDiffsPattern.FindStringSubmatch(fileContent)

	var blocks []string

	if len(keyDiffsMatch) > 1 {
		blocks = diffBlockPattern.FindAllString(keyDiffsMatch[1], -1)
	} else {
		// Fallback to old pattern if keyDiffs not found
		blocks = anyBlockPattern.FindAllString(fileContent, -1)
	}

	var allBindings []common.Binding

	for _, block := range blocks {
		// Check if block contains "added" (ignore "removed")
		if !strings.Contains(block, `["added"]`) {
			continue
		}

		// Extract action name
		nameMatch := namePattern.FindStringSubmatch(block)
		if len(nameMatch) < 2 {
			continue
		}
		actionName := nameMatch[1]

		// Extract complete "added" block
		addedBlockMatch := addedBlockPattern.FindStringSubmatch(block)
		if len(addedBlockMatch) < 2 {
			continue
		}
		addedContent := addedBlockMatch[1]

		// Extract all numbered entries [1] = { ... }, [2] = { ... }, etc.
		entryMatches := keyEntryPattern.FindAllStringSubmatch(addedContent, -1)

		if len(entryMatches) == 0 {
			continue
		}

		// Process each numbered entry separately
		for _, entryMatch := range entryMatches {
			if len(entryMatch) < 2 {
				continue
			}
			entryContent := entryMatch[1]

			// Extract the JOY key from this entry
			keyMatch := joyKeyPattern.FindStringSubmatch(entryContent)
			if len(keyMatch) < 2 {
				continue
			}

			binding := newJoyBinding(guid, deviceName, keyMatch[1], actionName)
			binding.Modifiers = []common.Modifier{}

			// Extract reformers from this entry if they exist
			reformersMatch := reformersPattern.FindStringSubmatch(entryContent)

			if len(reformersMatch) > 1 {
				reformersContent := reformersMatch[1]
				// Pattern to extract each modifier key: [1] = "VALUE"
				modMatches := numberedKeyPattern.FindAllStringSubmatch(reformersContent, -1)

				if len(modMatches) > 0 {
					for _, modMatch := range modMatches {
						if len(modMatch) > 1 {
							// Extract the modifier key and remove any prefix
							modKey := modMatch[1]
							if strings.HasPrefix(modKey, "Touche-") {
								modKey = strings.TrimPrefix(modKey, "Touche-")
							} else if strings.Contains(modKey, "-") {
								parts := strings.Split(modKey, "-")
								modKey = parts[len(parts)-1]
							}

							// Check if this is a switch using ModifierDeviceMap
							isSwitch := false
							if profile.ModifierDeviceMap != nil {
								if modInfo, exists := profile.ModifierDeviceMap[modKey]; exists {
									isSwitch = modInfo.IsSwitch
								}
							}

							modifier := common.Modifier{
								Keys:     []string{modKey},
								Action:   "",
								IsSwitch: isSwitch,
							}
							binding.Modifiers = append(binding.Modifiers, modifier)
						}
					}
				}
			}

			allBindings = append(allBindings, binding)
		}
	}

	return allBindings
}

// parseDCSLuaFile parses a DCS .lua file
func parseDCSLuaFile(filePath, fileName string, profile *common.Profile) error {
	// Extract GUID and file name
	// Format: {Something}_{GUID}.lua
	// The GUID is in the 36 characters before ".lua"

	// Simplified extraction of GUID and name
	// Case-insensitive pattern to support uppercase and lowercase
	guid := deviceGUIDPattern.FindString(fileName)

	if guid == "" {
		return fmt.Errorf("GUID not found in file name")
	}

	// Remove braces from GUID to match config format (without braces)
	guid = strings.Trim(guid, "{}")

	// Remove GUID and .lua to get the name
	deviceName := strings.TrimSuffix(fileName, ".lua")
	deviceName = strings.Replace(deviceName, "{"+guid+"}", "", 1) // Use braced version for removal
	deviceName = strings.Trim(deviceName, "_")
	// Remove " .diff" suffix if present (used in DCS file naming)
	deviceName = strings.TrimSuffix(deviceName, " .diff")

	// Add device if it doesn't exist already
	if _, exists := profile.Devices[guid]; !exists {
		profile.Devices[guid] = &common.Device{
			GUID: guid,
			Name: deviceName,
		}
	}

	// Read complete file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	fileContent := string(content)

	// Parse axis bindings
	parseAxisDiffs(fileContent, guid, deviceName, profile)

	// Parse button bindings
	allBindings := parseKeyDiffs(fileContent, guid, deviceName, profile)

	// Second pass: resolve modifier actions
	// Create a map: JOY_key -> action name
	keyToAction := make(map[string]string)
	for _, b := range allBindings {
		keyToAction[joyKeyFor(b)] = b.Action
	}

	// Now, resolve modifier actions
	for i := range allBindings {
		for j := range allBindings[i].Modifiers {
			modifier := &allBindings[i].Modifiers[j]
			if len(modifier.Keys) == 0 {
				continue
			}

			// The first key identifies the modifier; fall back to the key itself
			// when no binding defines an action for it.
			firstKey := modifier.Keys[0]
			if action, found := keyToAction[firstKey]; found {
				modifier.Action = action
			} else {
				modifier.Action = firstKey
			}
		}
	}

	// Note: Pure modifier bindings will be created later in createPureModifierBindings()
	// after all device files have been parsed

	// Add all bindings to profile
	profile.Bindings = append(profile.Bindings, allBindings...)

	// Parse keyboard bindings for Gremlins matching
	parseKeyboardBindingsFromLua(fileContent, profile)

	return nil
}

// parseDCSJoyInput parses a DCS joystick input (e.g. "JOY_BTN1", "JOY_X", "JOY_BTN_POV1_L")
func parseDCSJoyInput(input string) (common.InputType, string) {
	input = strings.TrimPrefix(input, "JOY_")

	// Hat switch with BTN prefix: BTN_POV1_U, BTN_POV1_L, etc.
	if strings.HasPrefix(input, "BTN_POV") {
		// Remove BTN_ prefix: BTN_POV1_L -> POV1_L
		input = strings.TrimPrefix(input, "BTN_")
		parts := strings.Split(input, "_")
		if len(parts) >= 2 {
			povNum := strings.TrimPrefix(parts[0], "POV")
			direction := ""
			if len(parts) > 1 {
				direction = parts[1]
			}
			return common.Hat, povNum + "_" + direction
		}
	}

	// Buttons: BTN1, BTN2, etc.
	if strings.HasPrefix(input, "BTN") {
		return common.Button, strings.TrimPrefix(input, "BTN")
	}

	// Hat without BTN prefix: POV1_U (rare but possible)
	if strings.Contains(input, "POV") {
		parts := strings.Split(input, "_")
		if len(parts) >= 2 {
			povNum := strings.TrimPrefix(parts[0], "POV")
			direction := ""
			if len(parts) > 1 {
				direction = parts[1]
			}
			return common.Hat, povNum + "_" + direction
		}
	}

	// Axes: X, Y, Z, RX, RY, RZ, SLIDER1, SLIDER2, etc.
	if strings.Contains(input, "SLIDER") {
		// Convert SLIDER1 -> SLIDER_1, SLIDER2 -> SLIDER_2, etc.
		if match := sliderPattern.FindStringSubmatch(input); len(match) > 1 {
			return common.Axis, fmt.Sprintf("SLIDER_%s", match[1])
		}
		return common.Axis, input
	}

	// Standard axes
	if standardAxisPattern.MatchString(input) {
		return common.Axis, input
	}

	// Default: consider as button
	return common.Button, input
}

// parseKeyboardBindingsFromLua extracts keyboard bindings from DCS LUA content
// This function looks for keyboard bindings in the same format as joystick bindings
// Keyboard keys are prefixed with "KEY_" in DCS
func parseKeyboardBindingsFromLua(fileContent string, profile *common.Profile) {
	// Extract keyDiffs block
	keyDiffsMatch := keyDiffsPattern.FindStringSubmatch(fileContent)

	if len(keyDiffsMatch) < 2 {
		// Try alternate pattern without "return diff"
		keyDiffsMatch = keyDiffsLoosePattern.FindStringSubmatch(fileContent)
		if len(keyDiffsMatch) < 2 {
			return
		}
	}

	keyDiffsContent := keyDiffsMatch[1]

	// Extract individual binding blocks
	blocks := diffBlockPattern.FindAllString(keyDiffsContent, -1)

	for _, block := range blocks {
		// Only process "added" bindings
		if !strings.Contains(block, `["added"]`) {
			continue
		}

		// Extract action name
		nameMatch := namePattern.FindStringSubmatch(block)
		if len(nameMatch) < 2 {
			continue
		}
		actionName := nameMatch[1]

		// Extract "added" block
		addedBlockMatch := addedBlockPattern.FindStringSubmatch(block)
		if len(addedBlockMatch) < 2 {
			continue
		}
		addedContent := addedBlockMatch[1]

		// Look for keyboard keys (KEY_* or simple keys like F1, G, etc.)
		keyMatches := anyKeyPattern.FindAllStringSubmatch(addedContent, -1)
		for _, keyMatch := range keyMatches {
			if len(keyMatch) < 2 {
				continue
			}
			keyValue := keyMatch[1]

			// Normalize key format to KEY_ prefix if not already present
			normalizedKey := keyValue
			if !strings.HasPrefix(keyValue, "KEY_") {
				normalizedKey = "KEY_" + keyValue
			}

			// Store as a special binding with "keyboard" as device
			// This will be used by Gremlins matching only
			binding := common.Binding{
				DeviceGUID:  "keyboard",
				DeviceName:  "Keyboard",
				InputType:   common.Button, // We treat keyboard keys as "buttons"
				InputID:     normalizedKey,
				Action:      normalizedKey,
				Description: actionName,
			}

			profile.Bindings = append(profile.Bindings, binding)
		}
	}
}

// parseDefaultKeyboardBindings parses the default DCS keyboard bindings
// These provide F1-F12 view commands and other defaults that aren't in user .diff files
func parseDefaultKeyboardBindings(profile *common.Profile) {
	// Common DCS installation paths
	possiblePaths := []string{
		"C:\\Program Files\\Eagle Dynamics\\DCS World\\Config\\Input\\Aircrafts\\Default\\keyboard\\default.lua",
		"C:\\Program Files (x86)\\Eagle Dynamics\\DCS World\\Config\\Input\\Aircrafts\\Default\\keyboard\\default.lua",
		"C:\\Program Files\\DCS World\\Config\\Input\\Aircrafts\\Default\\keyboard\\default.lua",
	}

	var defaultFilePath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			defaultFilePath = path
			break
		}
	}

	if defaultFilePath == "" {
		return
	}

	content, err := os.ReadFile(defaultFilePath)
	if err != nil {
		return
	}

	// Parse the default.lua file format: {combos = {{key = 'F1'}}, down = iCommandViewCockpit, name = _('F1 Cockpit view'), ...}
	// We're interested in simple key bindings (no reformers/modifiers)
	matches := defaultCombosPatern.FindAllStringSubmatch(string(content), -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		keyValue := match[1]
		actionName := match[2]

		// Only process F1-F12 keys for now (the main ones we care about for Gremlins)
		if !strings.HasPrefix(keyValue, "F") {
			continue
		}

		// Normalize key format
		normalizedKey := "KEY_" + keyValue

		binding := common.Binding{
			DeviceGUID:  "keyboard",
			DeviceName:  "Keyboard",
			InputType:   common.Button,
			InputID:     normalizedKey,
			Action:      normalizedKey,
			Description: actionName,
		}

		profile.Bindings = append(profile.Bindings, binding)
	}
}

// deviceRef identifies the device that owns a JOY_ key.
type deviceRef struct {
	guid string
	name string
}

// joyKeyFor rebuilds the DCS key a binding came from: "JOY_BTN24", "JOY_X",
// "JOY_1_U". It is the inverse of parseDCSJoyInput.
func joyKeyFor(b common.Binding) string {
	if b.InputType == common.Button {
		return "JOY_BTN" + b.InputID
	}
	return "JOY_" + b.InputID
}

// buildKeyToDeviceMap indexes the profile's bindings by JOY_ key so that a
// modifier or switch key can be traced back to the device carrying it.
func buildKeyToDeviceMap(profile *common.Profile) map[string]deviceRef {
	keyToDevice := make(map[string]deviceRef)
	for _, b := range profile.Bindings {
		keyToDevice[joyKeyFor(b)] = deviceRef{guid: b.DeviceGUID, name: b.DeviceName}
	}
	return keyToDevice
}

// modifierDeviceRef returns the device declared as owning key in modifiers.lua.
func modifierDeviceRef(profile *common.Profile, key string) (deviceRef, bool) {
	modInfo, exists := profile.ModifierDeviceMap[key]
	if !exists {
		return deviceRef{}, false
	}
	return deviceRef{guid: modInfo.DeviceGUID, name: modInfo.DeviceName}, true
}

// appendKeyRoleBinding records that key acts as a modifier or a switch ("Modifier"
// or "Switch" in keyType), attributing it to owner. modifierNum drives the colour
// coding and is normally assigned later by AssignModifierNumbers.
func appendKeyRoleBinding(profile *common.Profile, key, keyType string, owner deviceRef, modifierNum int) {
	inputType, inputID := parseDCSJoyInput(key)

	// Format: "Modifier BTN24" / "Switch BTN105", or "Modifier: <key>" for non-button keys
	displayText := fmt.Sprintf("%s: %s", keyType, key)
	if btn, ok := strings.CutPrefix(key, "JOY_BTN"); ok {
		displayText = fmt.Sprintf("%s BTN%s", keyType, btn)
	}

	profile.Bindings = append(profile.Bindings, common.Binding{
		DeviceGUID:  owner.guid,
		DeviceName:  owner.name,
		InputType:   inputType,
		InputID:     inputID,
		Action:      displayText,
		Modifiers:   []common.Modifier{},
		ModifierNum: modifierNum,
		ModifierKey: key, // Mark this as a modifier definition
	})
}

// hasBindingOn reports whether the profile already carries a binding on key for
// deviceGUID that satisfies match.
func hasBindingOn(profile *common.Profile, key, deviceGUID string, match func(common.Binding) bool) bool {
	for _, b := range profile.Bindings {
		if joyKeyFor(b) == key && b.DeviceGUID == deviceGUID && match(b) {
			return true
		}
	}
	return false
}

// collectUsedModifierKeys returns the keys actually referenced as modifiers by at
// least one action binding. Modifier definitions themselves are skipped.
func collectUsedModifierKeys(profile *common.Profile) map[string]bool {
	used := make(map[string]bool)
	for _, b := range profile.Bindings {
		if strings.HasPrefix(b.Action, "Modifier") {
			continue
		}
		for _, mod := range b.Modifiers {
			for _, key := range mod.Keys {
				used[key] = true
			}
		}
	}
	return used
}

// createPureModifierBindings creates bindings for buttons used only as modifiers
// This is called after all device files have been parsed to ensure we can find the correct device owner
func createPureModifierBindings(profile *common.Profile) {
	keyToDevice := buildKeyToDeviceMap(profile)

	isModifierBinding := func(b common.Binding) bool { return strings.HasPrefix(b.Action, "Modifier") }

	for modKey := range collectUsedModifierKeys(profile) {
		declared, declaredOK := modifierDeviceRef(profile, modKey)

		// Switches are handled by createPureSwitchBindings
		if modInfo, exists := profile.ModifierDeviceMap[modKey]; exists && modInfo.IsSwitch {
			continue
		}

		// Prefer the device declared in modifiers.lua, fall back to the device
		// carrying an existing binding on that key.
		owner := declared
		if !declaredOK || owner.guid == "" || owner.name == "" {
			var found bool
			if owner, found = keyToDevice[modKey]; !found {
				continue
			}
		}

		if !hasBindingOn(profile, modKey, declared.guid, isModifierBinding) {
			appendKeyRoleBinding(profile, modKey, "Modifier", owner, 0)
		}
	}
}

// createPureSwitchBindings creates bindings for buttons used only as switches
// This is called after all device files have been parsed to ensure we can find the correct device owner
func createPureSwitchBindings(profile *common.Profile) {
	isPlainAction := func(b common.Binding) bool {
		return !strings.HasPrefix(b.Action, "Switch") && !strings.HasPrefix(b.Action, "Modifier")
	}

	// Switch keys are always declared in modifiers.lua, so their owning device is
	// always known - no fallback to the bindings index is needed here.
	for switchKey, modInfo := range profile.ModifierDeviceMap {
		if !modInfo.IsSwitch {
			continue
		}

		// A switch that already has a real action on its own device needs no
		// dedicated "Switch" binding.
		if hasBindingOn(profile, switchKey, modInfo.DeviceGUID, isPlainAction) {
			continue
		}

		owner := deviceRef{guid: modInfo.DeviceGUID, name: modInfo.DeviceName}
		appendKeyRoleBinding(profile, switchKey, "Switch", owner, 0)
	}
}

// mergeModifierDeviceMapsAndCreateBindings merges ModifierDeviceMaps from all profiles
// and creates pure modifier/switch bindings for each profile using the merged map.
// This ensures that modifiers defined in one profile can be used in another profile.
func mergeModifierDeviceMapsAndCreateBindings(profiles []*common.Profile) {
	// Merge all ModifierDeviceMaps into one global map
	globalModifierDeviceMap := make(map[string]common.ModifierInfo)

	for _, profile := range profiles {
		if profile.ModifierDeviceMap != nil {
			for key, modInfo := range profile.ModifierDeviceMap {
				// If key already exists, keep the first one (they should be the same device anyway)
				if _, exists := globalModifierDeviceMap[key]; !exists {
					globalModifierDeviceMap[key] = modInfo
				}
			}
		}
	}

	// Set the merged map on each profile
	for _, profile := range profiles {
		// Keep the original profile's map but add entries from global map
		if profile.ModifierDeviceMap == nil {
			profile.ModifierDeviceMap = make(map[string]common.ModifierInfo)
		}

		// Merge global map into this profile's map
		for key, modInfo := range globalModifierDeviceMap {
			if _, exists := profile.ModifierDeviceMap[key]; !exists {
				profile.ModifierDeviceMap[key] = modInfo
			}
		}

		// Now create pure modifier and switch bindings for this profile
		// using the merged ModifierDeviceMap
		createPureModifierBindings(profile)
		createPureSwitchBindings(profile)
	}
}
