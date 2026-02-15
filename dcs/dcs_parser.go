package dcs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"simdiag/common"
	"strings"
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

// GetType implements SimulatorParser.GetType
func (p *Parser) GetType() common.SimulationType {
	return common.DCSWorld
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
		return nil, fmt.Errorf("Config/Input folder does not exist in: %s", basePath)
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
			fmt.Printf("  Avertissement: erreur lors du parsing de %s: %v\n", entry.Name(), err)
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

	// Pattern to match each modifier entry
	// ["Touche-JOY_BTN102"] = {
	//     ["device"] = "WINWING ... {530648C0-98C7-11f0-8002-444553540000}",
	//     ["key"] = "JOY_BTN102",
	//     ["switch"] = true,
	// },
	modifierPattern := regexp.MustCompile(`(?s)\["([^"]+)"\]\s*=\s*\{[^}]*\["device"\]\s*=\s*"([^"]*\{([^}]+)\})"[^}]*\["key"\]\s*=\s*"([^"]+)"[^}]*\["switch"\]\s*=\s*(true|false)[^}]*\}`)
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

// parseAxisDiffs extracts axis bindings from axisDiffs section
func parseAxisDiffs(fileContent, guid, deviceName string, profile *common.Profile) {
	// Find the start of axisDiffs block
	axisDiffsStartPattern := regexp.MustCompile(`\["axisDiffs"\]\s*=\s*\{`)
	axisDiffsStartMatch := axisDiffsStartPattern.FindStringIndex(fileContent)

	var axisDiffsContent string
	if axisDiffsStartMatch != nil {
		// Find the opening brace position
		bracePos := strings.Index(fileContent[axisDiffsStartMatch[0]:], "{")
		if bracePos >= 0 {
			actualBracePos := axisDiffsStartMatch[0] + bracePos
			// Extract balanced braces content
			content, endPos := extractBalancedBraces(fileContent, actualBracePos)
			if endPos > 0 {
				axisDiffsContent = content
			}
		}
	}

	if axisDiffsContent != "" {
		// Extract each axis block: ["a2001cdnil"] = { ... }
		var axisBlocks []string
		axisStartPattern := regexp.MustCompile(`\["a\d+[^"]*"\]\s*=\s*\{`)
		matches := axisStartPattern.FindAllStringIndex(axisDiffsContent, -1)

		for _, match := range matches {
			// Find the opening brace
			bracePos := strings.Index(axisDiffsContent[match[0]:], "{")
			if bracePos >= 0 {
				actualBracePos := match[0] + bracePos
				// Extract the complete axis block
				blockContent, endPos := extractBalancedBraces(axisDiffsContent, actualBracePos)
				if endPos > 0 {
					// Build the full block: key name + "= {" + content + "}"
					keyPart := axisDiffsContent[match[0]:actualBracePos]
					fullBlock := keyPart + "{" + blockContent + "}"
					axisBlocks = append(axisBlocks, fullBlock)
				}
			}
		}

		axisNamePattern := regexp.MustCompile(`\["name"\]\s*=\s*"([^"]*)"`)
		axisKeyPattern := regexp.MustCompile(`\["key"\]\s*=\s*"(JOY_[^"]+)"`)

		for _, axisBlock := range axisBlocks {
			// Extract action name
			nameMatch := axisNamePattern.FindStringSubmatch(axisBlock)
			if len(nameMatch) < 2 {
				continue
			}
			actionName := nameMatch[1]

			// Process "changed" or "added" axes (active bindings), ignore "removed"
			var bindingContent string

			changedStartPattern := regexp.MustCompile(`\["changed"\]\s*=\s*\{`)
			addedStartPattern := regexp.MustCompile(`\["added"\]\s*=\s*\{`)

			changedMatch := changedStartPattern.FindStringIndex(axisBlock)
			addedMatch := addedStartPattern.FindStringIndex(axisBlock)

			if changedMatch != nil {
				bracePos := strings.Index(axisBlock[changedMatch[0]:], "{")
				if bracePos >= 0 {
					actualBracePos := changedMatch[0] + bracePos
					content, _ := extractBalancedBraces(axisBlock, actualBracePos)
					if content != "" {
						bindingContent = content
					}
				}
			} else if addedMatch != nil {
				bracePos := strings.Index(axisBlock[addedMatch[0]:], "{")
				if bracePos >= 0 {
					actualBracePos := addedMatch[0] + bracePos
					content, _ := extractBalancedBraces(axisBlock, actualBracePos)
					if content != "" {
						bindingContent = content
					}
				}
			}

			if bindingContent == "" {
				continue // Skip removed/unchanged axes
			}

			// Extract all numbered entries [1] = { ... }, [2] = { ... }, etc.
			entryPattern := regexp.MustCompile(`(?s)\[\d+\]\s*=\s*\{([^{}]*(?:\{[^{}]*(?:\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}[^{}]*)*\}[^{}]*)*)\}`)
			entryMatches := entryPattern.FindAllStringSubmatch(bindingContent, -1)

			if len(entryMatches) == 0 {
				// Fallback: try to extract key directly if no numbered entries found
				keyMatch := axisKeyPattern.FindStringSubmatch(bindingContent)
				if len(keyMatch) >= 2 {
					joyKey := keyMatch[1]
					inputType, inputID := parseDCSJoyInput(joyKey)

					// Add (OFF) suffix to action if this is an OFF binding
					displayAction := actionName
					if strings.HasSuffix(inputID, "_OFF") {
						displayAction = actionName + " (OFF)"
					}

					binding := common.Binding{
						DeviceGUID: guid,
						DeviceName: deviceName,
						InputType:  inputType,
						InputID:    inputID,
						Action:     displayAction,
					}

					profile.Bindings = append(profile.Bindings, binding)
				}
				continue
			}

			// Process each numbered entry separately
			for _, entryMatch := range entryMatches {
				if len(entryMatch) < 2 {
					continue
				}
				entryContent := entryMatch[1]

				// Extract JOY key from this entry
				keyMatch := axisKeyPattern.FindStringSubmatch(entryContent)
				if len(keyMatch) < 2 {
					continue
				}
				joyKey := keyMatch[1]

				inputType, inputID := parseDCSJoyInput(joyKey)

				// Add (OFF) suffix to action if this is an OFF binding
				displayAction := actionName
				if strings.HasSuffix(inputID, "_OFF") {
					displayAction = actionName + " (OFF)"
				}

				binding := common.Binding{
					DeviceGUID: guid,
					DeviceName: deviceName,
					InputType:  inputType,
					InputID:    inputID,
					Action:     displayAction,
				}

				profile.Bindings = append(profile.Bindings, binding)
			}
		}
	}
}

// parseKeyDiffs extracts button bindings from keyDiffs section
func parseKeyDiffs(fileContent, guid, deviceName string, profile *common.Profile) []common.Binding {
	// First, extract the ["keyDiffs"] block
	keyDiffsPattern := regexp.MustCompile(`(?s)\["keyDiffs"\]\s*=\s*\{(.*)\},?\s*\}\s*return diff`)
	keyDiffsMatch := keyDiffsPattern.FindStringSubmatch(fileContent)

	var blocks []string

	if len(keyDiffsMatch) > 1 {
		keyDiffsContent := keyDiffsMatch[1]

		// Now extract each individual binding block from keyDiffs content
		blockPattern := regexp.MustCompile(`(?s)\["d[^"]+"\]\s*=\s*\{(?:[^{}]|\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*\})*\}`)
		blocks = blockPattern.FindAllString(keyDiffsContent, -1)
	} else {
		// Fallback to old pattern if keyDiffs not found
		blockPattern := regexp.MustCompile(`(?s)\["[^"]+"\]\s*=\s*\{(?:[^{}]|\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*\})*\}`)
		blocks = blockPattern.FindAllString(fileContent, -1)
	}

	// Pattern to extract action name
	namePattern := regexp.MustCompile(`\["name"\]\s*=\s*"([^"]+)"`)
	// Pattern to extract complete "added" block with all nested structures
	addedBlockPattern := regexp.MustCompile(`(?s)\["added"\]\s*=\s*\{((?:[^{}]|\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*\})*)\}`)

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
		entryPattern := regexp.MustCompile(`(?s)\[\d+\]\s*=\s*\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}`)
		entryMatches := entryPattern.FindAllStringSubmatch(addedContent, -1)

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
			keyPattern := regexp.MustCompile(`\["key"\]\s*=\s*"(JOY_[^"]+)"`)
			keyMatch := keyPattern.FindStringSubmatch(entryContent)
			if len(keyMatch) < 2 {
				continue
			}
			joyKey := keyMatch[1]

			inputType, inputID := parseDCSJoyInput(joyKey)

			// Add (OFF) suffix to action if this is an OFF binding
			displayAction := actionName
			if strings.HasSuffix(inputID, "_OFF") {
				displayAction = actionName + " (OFF)"
			}

			binding := common.Binding{
				DeviceGUID:  guid,
				DeviceName:  deviceName,
				InputType:   inputType,
				InputID:     inputID,
				Action:      displayAction,
				Description: "",
				Modifiers:   []common.Modifier{},
			}

			// Extract reformers from this entry if they exist
			reformersPattern := regexp.MustCompile(`(?s)\["reformers"\]\s*=\s*\{(.*?)\}`)
			reformersMatch := reformersPattern.FindStringSubmatch(entryContent)

			if len(reformersMatch) > 1 {
				reformersContent := reformersMatch[1]
				// Pattern to extract each modifier key: [1] = "VALUE"
				modKeyPattern := regexp.MustCompile(`\[\d+\]\s*=\s*"([^"]+)"`)
				modMatches := modKeyPattern.FindAllStringSubmatch(reformersContent, -1)

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
	guidPattern := regexp.MustCompile(`(?i)\{[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}\}`)
	guid := guidPattern.FindString(fileName)

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
		// Construire la clé JOY complète
		var joyKey string
		switch b.InputType {
		case common.Button:
			joyKey = "JOY_BTN" + b.InputID
		case common.Axis:
			joyKey = "JOY_" + b.InputID
		case common.Hat:
			// For HATs, build the key
			joyKey = "JOY_" + b.InputID
		}
		keyToAction[joyKey] = b.Action
	}

	// Collect all keys used as modifiers
	modifierKeys := make(map[string]bool)
	for i := range allBindings {
		for _, modifier := range allBindings[i].Modifiers {
			for _, key := range modifier.Keys {
				modifierKeys[key] = true
			}
		}
	}

	// Now, resolve modifier actions
	for i := range allBindings {
		for j := range allBindings[i].Modifiers {
			modifier := &allBindings[i].Modifiers[j]
			// Résoudre chaque clé
			if len(modifier.Keys) > 0 {
				// Prendre la première clé pour trouver l'action
				firstKey := modifier.Keys[0]
				if action, found := keyToAction[firstKey]; found {
					modifier.Action = action
				} else {
					// Si on ne trouve pas l'action, utiliser la clé elle-même
					modifier.Action = firstKey
				}
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
		sliderPattern := regexp.MustCompile(`^SLIDER(\d+)$`)
		if match := sliderPattern.FindStringSubmatch(input); len(match) > 1 {
			return common.Axis, fmt.Sprintf("SLIDER_%s", match[1])
		}
		return common.Axis, input
	}

	// Standard axes
	if matched, _ := regexp.MatchString(`^R?[XYZ]$`, input); matched {
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
	keyDiffsPattern := regexp.MustCompile(`(?s)\["keyDiffs"\]\s*=\s*\{(.*)\},?\s*\}\s*return diff`)
	keyDiffsMatch := keyDiffsPattern.FindStringSubmatch(fileContent)

	if len(keyDiffsMatch) < 2 {
		// Try alternate pattern without "return diff"
		keyDiffsPattern2 := regexp.MustCompile(`(?s)\["keyDiffs"\]\s*=\s*\{(.*)\}`)
		keyDiffsMatch = keyDiffsPattern2.FindStringSubmatch(fileContent)
		if len(keyDiffsMatch) < 2 {
			return
		}
	}

	keyDiffsContent := keyDiffsMatch[1]

	// Extract individual binding blocks
	blockPattern := regexp.MustCompile(`(?s)\["d[^"]+"\]\s*=\s*\{(?:[^{}]|\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*\})*\}`)
	blocks := blockPattern.FindAllString(keyDiffsContent, -1)

	namePattern := regexp.MustCompile(`\["name"\]\s*=\s*"([^"]+)"`)
	addedBlockPattern := regexp.MustCompile(`(?s)\["added"\]\s*=\s*\{((?:[^{}]|\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*\})*)\}`)
	keyPattern := regexp.MustCompile(`\["key"\]\s*=\s*"([^"]+)"`) // Accept any key format, not just KEY_

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
		keyMatches := keyPattern.FindAllStringSubmatch(addedContent, -1)
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
	pattern := regexp.MustCompile(`\{combos\s*=\s*\{\{key\s*=\s*'([^']+)'\}\},\s*down\s*=\s*[^,]+,\s*name\s*=\s*_\('([^']+)'\)`)
	matches := pattern.FindAllStringSubmatch(string(content), -1)

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

// createPureKeyBinding creates a binding for a modifier or switch key
func createPureKeyBinding(profile *common.Profile, key, keyType string, keyToDevice map[string]struct{ guid, name string }) {
	// Try to get device info from ModifierDeviceMap first
	var deviceInfo struct {
		guid string
		name string
	}
	found := false

	if profile.ModifierDeviceMap != nil {
		if modInfo, exists := profile.ModifierDeviceMap[key]; exists {
			deviceInfo = struct {
				guid string
				name string
			}{guid: modInfo.DeviceGUID, name: modInfo.DeviceName}
			found = true
		}
	}

	// Fallback to keyToDevice if not found in ModifierDeviceMap
	if !found {
		deviceInfo, found = keyToDevice[key]
	}

	if found {
		// Found the device owner, create the binding
		inputType, inputID := parseDCSJoyInput(key)
		// Format: "Modifier BTN80" or "Switch BTN105"
		var displayText string
		if strings.HasPrefix(key, "JOY_BTN") {
			displayText = fmt.Sprintf("%s BTN%s", keyType, strings.TrimPrefix(key, "JOY_BTN"))
		} else {
			displayText = fmt.Sprintf("%s: %s", keyType, key)
		}
		binding := common.Binding{
			DeviceGUID:  deviceInfo.guid,
			DeviceName:  deviceInfo.name,
			InputType:   inputType,
			InputID:     inputID,
			Action:      displayText,
			Modifiers:   []common.Modifier{},
			ModifierKey: key, // Mark this as a modifier definition
		}
		profile.Bindings = append(profile.Bindings, binding)
	}
}

// createModifierBinding creates a modifier binding with the specified ModifierNum for color coding
func createModifierBinding(profile *common.Profile, key string, modifierNum int, deviceGUID, deviceName string, keyToDevice map[string]struct{ guid, name string }) {
	// Try to get device info from parameters first, then fallback to keyToDevice
	var deviceInfo struct {
		guid string
		name string
	}
	found := false

	if deviceGUID != "" && deviceName != "" {
		deviceInfo = struct {
			guid string
			name string
		}{guid: deviceGUID, name: deviceName}
		found = true
	} else {
		deviceInfo, found = keyToDevice[key]
	}

	if found {
		// Found the device owner, create the binding
		inputType, inputID := parseDCSJoyInput(key)
		// Format: "Modifier BTN24" (the number will be assigned later by AssignModifierNumbers)
		var displayText string
		if strings.HasPrefix(key, "JOY_BTN") {
			displayText = fmt.Sprintf("Modifier BTN%s", strings.TrimPrefix(key, "JOY_BTN"))
		} else {
			displayText = fmt.Sprintf("Modifier: %s", key)
		}

		binding := common.Binding{
			DeviceGUID:  deviceInfo.guid,
			DeviceName:  deviceInfo.name,
			InputType:   inputType,
			InputID:     inputID,
			Action:      displayText,
			Modifiers:   []common.Modifier{},
			ModifierNum: modifierNum, // Set the ModifierNum for color coding
			ModifierKey: key,         // Mark this as a modifier definition
		}
		profile.Bindings = append(profile.Bindings, binding)
	}
}

// createPureModifierBindings creates bindings for buttons used only as modifiers
// This is called after all device files have been parsed to ensure we can find the correct device owner
func createPureModifierBindings(profile *common.Profile) {
	// Collect all keys defined as modifiers (not switches) in modifiers.lua
	modifierKeys := make(map[string]bool)

	// Get all modifiers from ModifierDeviceMap (from modifiers.lua)
	if profile.ModifierDeviceMap != nil {
		for key, modInfo := range profile.ModifierDeviceMap {
			if !modInfo.IsSwitch {
				modifierKeys[key] = true
			}
		}
	}

	// Build a map: JOY_key -> device info (from all existing bindings)
	keyToDevice := make(map[string]struct {
		guid string
		name string
	})

	for _, b := range profile.Bindings {
		var joyKey string
		switch b.InputType {
		case common.Button:
			joyKey = "JOY_BTN" + b.InputID
		case common.Axis:
			joyKey = "JOY_" + b.InputID
		case common.Hat:
			joyKey = "JOY_" + b.InputID
		}

		if joyKey != "" {
			keyToDevice[joyKey] = struct {
				guid string
				name string
			}{guid: b.DeviceGUID, name: b.DeviceName}
		}
	}

	// Collect all modifiers that are ACTUALLY USED in bindings
	usedModifiers := make(map[string]bool)

	// Scan all bindings to find which keys are used as modifiers
	for _, b := range profile.Bindings {
		// Skip bindings that are themselves modifier definitions
		if strings.HasPrefix(b.Action, "Modifier") {
			continue
		}

		// If this binding has modifiers, record them
		if len(b.Modifiers) > 0 {
			for _, mod := range b.Modifiers {
				for _, key := range mod.Keys {
					usedModifiers[key] = true
				}
			}
		}
	}

	// Create bindings for modifiers that are ACTUALLY USED
	// ModifierNum will be assigned later by AssignModifierNumbers()
	for modKey := range usedModifiers {
		// Skip if this is actually a switch (not a modifier)
		if profile.ModifierDeviceMap != nil {
			if modInfo, exists := profile.ModifierDeviceMap[modKey]; exists && modInfo.IsSwitch {
				continue // This is a switch, will be handled by createPureSwitchBindings
			}
		}

		// Get the device info for this modifier key from ModifierDeviceMap
		var expectedDeviceGUID string
		var expectedDeviceName string
		if profile.ModifierDeviceMap != nil {
			if modInfo, exists := profile.ModifierDeviceMap[modKey]; exists {
				expectedDeviceGUID = modInfo.DeviceGUID
				expectedDeviceName = modInfo.DeviceName
			}
		}

		// Check if a "Modifier" binding already exists for this key on this device
		hasModifierBinding := false
		for _, b := range profile.Bindings {
			var joyKey string
			switch b.InputType {
			case common.Button:
				joyKey = "JOY_BTN" + b.InputID
			case common.Axis:
				joyKey = "JOY_" + b.InputID
			case common.Hat:
				joyKey = "JOY_" + b.InputID
			}

			// Check if this is already a "Modifier" binding for this key and device
			if joyKey == modKey && b.DeviceGUID == expectedDeviceGUID && strings.HasPrefix(b.Action, "Modifier") {
				hasModifierBinding = true
				break
			}
		}

		// If no "Modifier" binding exists yet, create one
		// ModifierNum will be assigned later by AssignModifierNumbers()
		if !hasModifierBinding {
			createModifierBinding(profile, modKey, 0, expectedDeviceGUID, expectedDeviceName, keyToDevice)
		}
	}
}

// createPureSwitchBindings creates bindings for buttons used only as switches
// This is called after all device files have been parsed to ensure we can find the correct device owner
func createPureSwitchBindings(profile *common.Profile) {
	// Collect all keys defined as switches in modifiers.lua
	switchKeys := make(map[string]bool)

	// Get all switches from ModifierDeviceMap (from modifiers.lua)
	if profile.ModifierDeviceMap != nil {
		for key, modInfo := range profile.ModifierDeviceMap {
			if modInfo.IsSwitch {
				switchKeys[key] = true
			}
		}
	}

	// Build a map: JOY_key -> device info (from all existing bindings)
	keyToDevice := make(map[string]struct {
		guid string
		name string
	})

	for _, b := range profile.Bindings {
		var joyKey string
		switch b.InputType {
		case common.Button:
			joyKey = "JOY_BTN" + b.InputID
		case common.Axis:
			joyKey = "JOY_" + b.InputID
		case common.Hat:
			joyKey = "JOY_" + b.InputID
		}

		if joyKey != "" {
			keyToDevice[joyKey] = struct {
				guid string
				name string
			}{guid: b.DeviceGUID, name: b.DeviceName}
		}
	}

	// Create bindings for pure switches (used as switch but has no action binding)
	for switchKey := range switchKeys {
		// Get the device info for this switch key from ModifierDeviceMap
		var expectedDeviceGUID string
		if profile.ModifierDeviceMap != nil {
			if modInfo, exists := profile.ModifierDeviceMap[switchKey]; exists {
				expectedDeviceGUID = modInfo.DeviceGUID
			}
		}

		// Check if this key already has an action binding on the SAME device
		hasAction := false
		for _, b := range profile.Bindings {
			var joyKey string
			switch b.InputType {
			case common.Button:
				joyKey = "JOY_BTN" + b.InputID
			case common.Axis:
				joyKey = "JOY_" + b.InputID
			case common.Hat:
				joyKey = "JOY_" + b.InputID
			}

			// Check both key AND device match
			if joyKey == switchKey && b.DeviceGUID == expectedDeviceGUID && !strings.HasPrefix(b.Action, "Switch") && !strings.HasPrefix(b.Action, "Modifier") {
				hasAction = true
				break
			}
		}

		// If no action binding exists, create a "pure switch" binding
		if !hasAction {
			createPureKeyBinding(profile, switchKey, "Switch", keyToDevice)
		}
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
