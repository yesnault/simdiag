package il2

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"simdiag/common"
	"strings"
)

// Parser implements the SimulatorParser interface for IL-2 Sturmovik
type Parser struct{}

// NewParser creates a new IL-2 parser instance
func NewParser() *Parser {
	return &Parser{}
}

// Parse implements SimulatorParser.Parse
func (p *Parser) Parse(basePath string) (*common.ProfileCollection, error) {
	return parseIL2(basePath)
}

// GetName implements SimulatorParser.GetName
func (p *Parser) GetName() string {
	return "IL-2 Sturmovik"
}

// GetType implements SimulatorParser.GetType
func (p *Parser) GetType() common.SimulationType {
	return common.IL2Sturmovik
}

// parseIL2 parses IL-2 Sturmovik configuration files
func parseIL2(basePath string) (*common.ProfileCollection, error) {
	collection := &common.ProfileCollection{
		Profiles: make([]*common.Profile, 0),
	}

	// Check that files exist
	globalActionsPath := filepath.Join(basePath, "global.actions")
	devicesPath := filepath.Join(basePath, "devices.txt")

	if _, err := os.Stat(globalActionsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("global.actions file not found in: %s", basePath)
	}

	if _, err := os.Stat(devicesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("devices.txt file not found in: %s", basePath)
	}

	// Parse devices.txt to get devices
	// Returns a map with composite keys "configID:GUID" for lookup during binding parsing
	devicesWithConfigID, err := parseIL2Devices(devicesPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing devices.txt: %w", err)
	}

	// Create a clean device map by GUID only (for the profile)
	devices := make(map[string]*common.Device)
	for _, device := range devicesWithConfigID {
		devices[device.GUID] = device
	}

	// Parse global.actions to get bindings
	profile := &common.Profile{
		Name:     "IL-2",
		SimType:  common.IL2Sturmovik,
		Devices:  devices,
		Bindings: make([]common.Binding, 0),
	}

	err = parseIL2GlobalActions(globalActionsPath, profile, devicesWithConfigID)
	if err != nil {
		return nil, fmt.Errorf("error parsing global.actions: %w", err)
	}

	collection.Profiles = append(collection.Profiles, profile)

	return collection, nil
}

// parseIL2Devices parses the devices.txt file
func parseIL2Devices(devicesPath string) (map[string]*common.Device, error) {
	devices := make(map[string]*common.Device)

	file, err := os.Open(devicesPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// IL-2 devices.txt format:
	// configId,guid,model|
	// Example: 1,%22b0c891c0-3f30-11f0-0000545345440380%22,T-Rudder|
	// The GUID and model are URL-encoded

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines, comments, and header
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "configId,") {
			continue
		}

		// Remove trailing | if present
		line = strings.TrimSuffix(line, "|")

		// Split by comma
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		configID := strings.TrimSpace(parts[0])
		guidEncoded := strings.TrimSpace(parts[1])
		modelEncoded := strings.TrimSpace(parts[2])

		// URL-decode the GUID and model
		guid, err := url.QueryUnescape(guidEncoded)
		if err != nil {
			fmt.Printf("Warning: failed to decode GUID on line %d: %v\n", lineNum, err)
			continue
		}

		model, err := url.QueryUnescape(modelEncoded)
		if err != nil {
			fmt.Printf("Warning: failed to decode model on line %d: %v\n", lineNum, err)
			continue
		}

		// Remove quotes from GUID if present
		guid = strings.Trim(guid, "\"")

		// Store the device using a composite key: configID:GUID
		// This allows us to look up by config ID later
		compositeKey := fmt.Sprintf("%s:%s", configID, guid)

		devices[compositeKey] = &common.Device{
			GUID: guid,
			Name: model,
		}
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no devices found in devices.txt")
	}

	return devices, scanner.Err()
}

// parseIL2GlobalActions parses the global.actions file
func parseIL2GlobalActions(actionsPath string, profile *common.Profile, devices map[string]*common.Device) error {
	// Read file as bytes first
	fileBytes, err := os.ReadFile(actionsPath)
	if err != nil {
		return err
	}

	// IL-2 files are mostly Windows-1252/ISO-8859-1 encoded, but contain some UTF-8 sequences
	// (like the right single quotation mark U+2019 = E2 80 99 in UTF-8)
	var content string
	for i := 0; i < len(fileBytes); i++ {
		// Check for UTF-8 encoded right single quotation mark (E2 80 99)
		if i+2 < len(fileBytes) && fileBytes[i] == 0xE2 && fileBytes[i+1] == 0x80 && fileBytes[i+2] == 0x99 {
			content += "'"
			i += 2 // Skip the next 2 bytes
		} else {
			// Treat as Windows-1252
			content += string(rune(fileBytes[i]))
		}
	}

	// First pass: build action descriptions mapping
	actionDescriptions := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines, comments, and header
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "&actions=") {
			continue
		}

		// Check if this line has a comment with description
		if !strings.Contains(line, "|") {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) > 1 {
			comment := strings.TrimSpace(parts[1])
			if strings.HasPrefix(comment, "//") {
				description := strings.TrimSpace(comment[2:])
				if description != "" {
					// Extract action name from the binding part
					bindingPart := parts[0]
					bindingFields := strings.Split(bindingPart, ",")
					if len(bindingFields) >= 1 {
						actionName := strings.TrimSpace(bindingFields[0])
						// Store description for this action (only if not already stored)
						if _, exists := actionDescriptions[actionName]; !exists {
							actionDescriptions[actionName] = description
						}
					}
				}
			}
		}
	}

	// Second pass: parse bindings
	scanner = bufio.NewScanner(strings.NewReader(content))

	// Patterns for different input types
	buttonPattern := regexp.MustCompile(`^joy(\d+)_b(\d+)$`)
	axisPattern := regexp.MustCompile(`^joy(\d+)_axis_([xyzwqsturp])$`)
	povPattern := regexp.MustCompile(`^joy(\d+)_pov(\d+)_(\d+)$`)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines, comments, and header
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "&actions=") {
			continue
		}

		// Split by | to separate binding from comment
		if !strings.Contains(line, "|") {
			continue
		}

		parts := strings.Split(line, "|")
		bindingPart := strings.TrimSpace(parts[0])

		// Parse binding part: action_name, device_ref, invert
		bindingFields := strings.Split(bindingPart, ",")
		if len(bindingFields) < 2 {
			continue
		}

		actionName := strings.TrimSpace(bindingFields[0])
		deviceRef := strings.TrimSpace(bindingFields[1])

		// Get description from our pre-built mapping
		description := actionDescriptions[actionName]
		if description == "" {
			description = actionName // Fallback to action name if no description found
		}

		// Handle keyboard bindings (for TARGET matching only, not for display)
		if strings.HasPrefix(deviceRef, "key_") {
			// Parse keyboard binding
			binding := common.Binding{
				DeviceGUID:  "keyboard",
				DeviceName:  "Keyboard",
				InputType:   common.Button, // Treat keyboard keys as buttons
				InputID:     deviceRef,
				Action:      actionName,
				Description: description,
			}
			profile.Bindings = append(profile.Bindings, binding)
			continue
		}

		// Skip mouse bindings (not used for TARGET matching)
		if strings.HasPrefix(deviceRef, "mouse_") {
			continue
		}

		// Handle multiple device references separated by /
		deviceRefs := strings.Split(deviceRef, "/")

		for _, ref := range deviceRefs {
			ref = strings.TrimSpace(ref)

			// Try to parse as button
			if matches := buttonPattern.FindStringSubmatch(ref); matches != nil {
				joyNum := matches[1]
				btnNum := matches[2]

				// Find device GUID for this joy number
				deviceGUID := findDeviceGUIDByConfigID(devices, joyNum)
				if deviceGUID == "" {
					fmt.Printf("  [IL2] Warning: No device found for joy%s (button binding)\n", joyNum)
					continue
				}

				compositeKey := fmt.Sprintf("%s:%s", joyNum, deviceGUID)
				device := devices[compositeKey]
				if device == nil {
					fmt.Printf("  [IL2] Warning: Device is nil for key '%s'\n", compositeKey)
					continue
				}

				// IL-2 button numbers are 0-based, we use 1-based
				btnNumInt := 0
				_, _ = fmt.Sscanf(btnNum, "%d", &btnNumInt)
				btnNumInt++ // Convert to 1-based

				binding := common.Binding{
					DeviceGUID:  deviceGUID,
					DeviceName:  device.Name,
					InputType:   common.Button,
					InputID:     fmt.Sprintf("%d", btnNumInt),
					Action:      actionName,
					Description: description,
				}

				profile.Bindings = append(profile.Bindings, binding)
				continue
			}

			// Try to parse as axis
			if matches := axisPattern.FindStringSubmatch(ref); matches != nil {
				joyNum := matches[1]
				axisLetter := strings.ToLower(matches[2])

				// Find device GUID for this joy number
				deviceGUID := findDeviceGUIDByConfigID(devices, joyNum)
				if deviceGUID == "" {
					continue
				}

				compositeKey := fmt.Sprintf("%s:%s", joyNum, deviceGUID)
				device := devices[compositeKey]
				if device == nil {
					continue
				}

				// Map IL-2 axis letters to DirectInput axis names
				axisMapping := map[string]string{
					"x": "X",
					"y": "Y",
					"z": "Z",
					"w": "RX",
					"s": "RY",
					"t": "RZ",
					"q": "SLIDER_2",
					"p": "SLIDER_1",
					"u": "U",
					"r": "V",
				}

				axisID := axisMapping[axisLetter]
				if axisID == "" {
					axisID = strings.ToUpper(axisLetter)
				}

				binding := common.Binding{
					DeviceGUID:  deviceGUID,
					DeviceName:  device.Name,
					InputType:   common.Axis,
					InputID:     axisID,
					Action:      actionName,
					Description: description,
				}

				profile.Bindings = append(profile.Bindings, binding)
				continue
			}

			// Try to parse as POV/HAT
			if matches := povPattern.FindStringSubmatch(ref); matches != nil {
				joyNum := matches[1]
				povNum := matches[2]
				direction := matches[3]

				// Find device GUID for this joy number
				deviceGUID := findDeviceGUIDByConfigID(devices, joyNum)
				if deviceGUID == "" {
					continue
				}

				compositeKey := fmt.Sprintf("%s:%s", joyNum, deviceGUID)
				device := devices[compositeKey]
				if device == nil {
					continue
				}

				// Convert POV number (0-based) to 1-based
				povNumInt := 0
				_, _ = fmt.Sscanf(povNum, "%d", &povNumInt)
				povNumInt++

				// Map direction angles to directional names
				directionMapping := map[string]string{
					"0":   "U",
					"45":  "UR",
					"90":  "R",
					"135": "DR",
					"180": "D",
					"225": "DL",
					"270": "L",
					"315": "UL",
				}

				directionName := directionMapping[direction]
				if directionName == "" {
					directionName = direction
				}

				hatDirection := fmt.Sprintf("%d_%s", povNumInt, directionName)

				binding := common.Binding{
					DeviceGUID:  deviceGUID,
					DeviceName:  device.Name,
					InputType:   common.Hat,
					InputID:     hatDirection,
					Action:      actionName,
					Description: description,
				}

				profile.Bindings = append(profile.Bindings, binding)
				continue
			}
		}
	}

	// Parse keyboard bindings for Gremlins matching
	parseKeyboardBindingsFromIL2(content, profile, actionDescriptions)

	return scanner.Err()
}

// findDeviceGUIDByConfigID finds a device GUID by its IL-2 config ID
func findDeviceGUIDByConfigID(devices map[string]*common.Device, configID string) string {
	// Devices are stored with composite keys "configID:GUID"
	// Find the one that starts with our configID
	for key := range devices {
		parts := strings.Split(key, ":")
		if len(parts) == 2 && parts[0] == configID {
			return parts[1] // Return the GUID part
		}
	}
	return ""
}

// parseKeyboardBindingsFromIL2 extracts keyboard bindings from IL-2 global.actions
// This parses the same file but captures keyboard bindings for Gremlins matching
// actionDescriptions maps action names to their human-readable descriptions
func parseKeyboardBindingsFromIL2(actionsContent string, profile *common.Profile, actionDescriptions map[string]string) {
	// IL-2 keyboard bindings format: action, key_KEYNAME, invert| // description
	// Example: ipdinc, key_lshift+key_add, 0| // +IPD correction

	scanner := bufio.NewScanner(strings.NewReader(actionsContent))

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines, comments, and header
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "&actions=") {
			continue
		}

		// Split by | to separate binding from comment
		if !strings.Contains(line, "|") {
			continue
		}

		parts := strings.Split(line, "|")
		bindingPart := strings.TrimSpace(parts[0])
		comment := ""
		if len(parts) > 1 {
			comment = strings.TrimSpace(parts[1])
			comment = strings.TrimPrefix(comment, "//")
			comment = strings.TrimSpace(comment)
		}

		// Parse binding part: action_name, device_ref, invert
		bindingFields := strings.Split(bindingPart, ",")
		if len(bindingFields) < 2 {
			continue
		}

		actionName := strings.TrimSpace(bindingFields[0])
		deviceRef := strings.TrimSpace(bindingFields[1])

		// Only process keyboard bindings (key_*)
		if !strings.HasPrefix(deviceRef, "key_") {
			continue
		}

		// Use comment as description, or look up in actionDescriptions, or fall back to action name
		description := comment
		if description == "" {
			// Try to get description from the pre-built mapping (from lines that have comments)
			if desc, found := actionDescriptions[actionName]; found && desc != "" {
				description = desc
			} else {
				description = actionName
			}
		}

		// IL-2 might have multiple keys separated by /
		keyNames := strings.Split(deviceRef, "/")

		for _, key := range keyNames {
			key = strings.TrimSpace(key)

			if key == "" {
				continue
			}

			// Store as a special binding with "keyboard" as device
			// Keep the full key combination (e.g., "key_lshift+key_add")
			binding := common.Binding{
				DeviceGUID:  "keyboard",
				DeviceName:  "Keyboard",
				InputType:   common.Button,
				InputID:     key,
				Action:      actionName,
				Description: description,
			}

			profile.Bindings = append(profile.Bindings, binding)
		}
	}
}
