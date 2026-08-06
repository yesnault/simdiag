package il2

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"simdiag/common"
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

// DecodeIL2Text converts raw IL-2 configuration bytes to a string.
// IL-2 files are mostly Windows-1252/ISO-8859-1 encoded, but contain some UTF-8 sequences
// (like the right single quotation mark U+2019 = E2 80 99 in UTF-8).
func DecodeIL2Text(fileBytes []byte) string {
	var sb strings.Builder
	for i := 0; i < len(fileBytes); i++ {
		// Check for UTF-8 encoded right single quotation mark (E2 80 99)
		if i+2 < len(fileBytes) && fileBytes[i] == 0xE2 && fileBytes[i+1] == 0x80 && fileBytes[i+2] == 0x99 {
			sb.WriteString("'")
			i += 2 // Skip the next 2 bytes
		} else {
			// Treat as Windows-1252
			sb.WriteRune(rune(fileBytes[i]))
		}
	}
	return sb.String()
}

// extractActionDescriptions builds the action name -> human readable description mapping
// from the "| // description" comments of a global.actions content.
func extractActionDescriptions(content string) map[string]string {
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

	return actionDescriptions
}

// LoadActionDescriptions reads a global.actions file and returns the action name ->
// description mapping built from its inline comments.
//
// IL-2 Korea ships no human readable labels, so its parser reuses this mapping from a
// configured Great Battles installation. Returns an empty map if the file cannot be read.
func LoadActionDescriptions(globalActionsPath string) map[string]string {
	fileBytes, err := os.ReadFile(globalActionsPath)
	if err != nil {
		return map[string]string{}
	}
	return extractActionDescriptions(DecodeIL2Text(fileBytes))
}

// Patterns for the device references found in global.actions
var (
	joyButtonPattern = regexp.MustCompile(`^joy(\d+)_b(\d+)$`)
	joyAxisPattern   = regexp.MustCompile(`^joy(\d+)_axis_([xyzwqsturp])$`)
	joyPovPattern    = regexp.MustCompile(`^joy(\d+)_pov(\d+)_(\d+)$`)
)

// actionLine holds one parsed line of global.actions.
type actionLine struct {
	action      string
	description string
	deviceRef   string
}

// parseActionLine splits a global.actions line into its action name, its
// human-readable description and its device reference. It reports false for
// blank lines, comments and malformed entries.
func parseActionLine(line string, actionDescriptions map[string]string) (actionLine, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" ||
		strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "&actions=") {
		return actionLine{}, false
	}

	// Everything after "|" is a comment
	bindingPart, _, found := strings.Cut(line, "|")
	if !found {
		return actionLine{}, false
	}

	// Binding part: action_name, device_ref, invert
	bindingFields := strings.Split(strings.TrimSpace(bindingPart), ",")
	if len(bindingFields) < 2 {
		return actionLine{}, false
	}

	action := strings.TrimSpace(bindingFields[0])

	description := actionDescriptions[action]
	if description == "" {
		description = action // Fallback to action name if no description found
	}

	return actionLine{
		action:      action,
		description: description,
		deviceRef:   strings.TrimSpace(bindingFields[1]),
	}, true
}

// lookupDevice resolves an IL-2 joy number to the device it refers to. warn
// enables the diagnostics printed for button bindings only, matching the
// historical output.
func lookupDevice(devices map[string]*common.Device, joyNum string, warn bool) (*common.Device, string) {
	deviceGUID := findDeviceGUIDByConfigID(devices, joyNum)
	if deviceGUID == "" {
		if warn {
			fmt.Printf("  [IL2] Warning: No device found for joy%s (button binding)\n", joyNum)
		}
		return nil, ""
	}

	compositeKey := fmt.Sprintf("%s:%s", joyNum, deviceGUID)
	device := devices[compositeKey]
	if device == nil {
		if warn {
			fmt.Printf("  [IL2] Warning: Device is nil for key '%s'\n", compositeKey)
		}
		return nil, ""
	}

	return device, deviceGUID
}

// bindingForRef converts a single IL-2 device reference (joy1_b12, joy1_axis_x,
// joy1_pov0_90) into a binding. It returns nil for unrecognised references and
// for references naming an unknown device.
func bindingForRef(ref string, line actionLine, devices map[string]*common.Device) *common.Binding {
	newBinding := func(device *common.Device, guid string, inputType common.InputType, inputID string) *common.Binding {
		return &common.Binding{
			DeviceGUID:  guid,
			DeviceName:  device.Name,
			InputType:   inputType,
			InputID:     inputID,
			Action:      line.action,
			Description: line.description,
		}
	}

	if matches := joyButtonPattern.FindStringSubmatch(ref); matches != nil {
		device, guid := lookupDevice(devices, matches[1], true)
		if device == nil {
			return nil
		}
		// IL-2 button numbers are 0-based, we use 1-based
		btnNum, _ := strconv.Atoi(matches[2])
		return newBinding(device, guid, common.Button, strconv.Itoa(btnNum+1))
	}

	if matches := joyAxisPattern.FindStringSubmatch(ref); matches != nil {
		device, guid := lookupDevice(devices, matches[1], false)
		if device == nil {
			return nil
		}
		return newBinding(device, guid, common.Axis, AxisLetterToAxisID(matches[2]))
	}

	if matches := joyPovPattern.FindStringSubmatch(ref); matches != nil {
		device, guid := lookupDevice(devices, matches[1], false)
		if device == nil {
			return nil
		}
		// POV numbers are 0-based too
		povNum, _ := strconv.Atoi(matches[2])
		hatDirection := fmt.Sprintf("%d_%s", povNum+1, PovAngleToDirection(matches[3]))
		return newBinding(device, guid, common.Hat, hatDirection)
	}

	return nil
}

// parseIL2GlobalActions parses the global.actions file
func parseIL2GlobalActions(actionsPath string, profile *common.Profile, devices map[string]*common.Device) error {
	// Read file as bytes first
	fileBytes, err := os.ReadFile(actionsPath)
	if err != nil {
		return err
	}

	content := DecodeIL2Text(fileBytes)

	// First pass: build action descriptions mapping
	actionDescriptions := extractActionDescriptions(content)

	// Second pass: parse bindings
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line, ok := parseActionLine(scanner.Text(), actionDescriptions)
		if !ok {
			continue
		}

		// Keyboard bindings are kept for TARGET matching only, never displayed
		if strings.HasPrefix(line.deviceRef, "key_") {
			profile.Bindings = append(profile.Bindings, common.Binding{
				DeviceGUID:  "keyboard",
				DeviceName:  "Keyboard",
				InputType:   common.Button, // Treat keyboard keys as buttons
				InputID:     line.deviceRef,
				Action:      line.action,
				Description: line.description,
			})
			continue
		}

		// Skip mouse bindings (not used for TARGET matching)
		if strings.HasPrefix(line.deviceRef, "mouse_") {
			continue
		}

		// One action can be bound on several devices, separated by /
		for _, ref := range strings.Split(line.deviceRef, "/") {
			if binding := bindingForRef(strings.TrimSpace(ref), line, devices); binding != nil {
				profile.Bindings = append(profile.Bindings, *binding)
			}
		}
	}

	// Parse keyboard bindings for Gremlins matching
	parseKeyboardBindingsFromIL2(content, profile, actionDescriptions)

	return scanner.Err()
}

// axisLetterToAxisID maps IL-2 axis letters to DirectInput axis names.
// Shared by both IL-2 engines (Great Battles and Korea use the same letters).
var axisLetterToAxisID = map[string]string{
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

// povAngleToDirection maps POV/hat angles to directional names.
// Shared by both IL-2 engines.
var povAngleToDirection = map[string]string{
	"0":   "U",
	"45":  "UR",
	"90":  "R",
	"135": "DR",
	"180": "D",
	"225": "DL",
	"270": "L",
	"315": "UL",
}

// AxisLetterToAxisID converts an IL-2 axis letter (e.g. "w") to a DirectInput axis
// name (e.g. "RX"). Unknown letters are returned uppercased.
func AxisLetterToAxisID(letter string) string {
	if axisID, ok := axisLetterToAxisID[strings.ToLower(letter)]; ok {
		return axisID
	}
	return strings.ToUpper(letter)
}

// PovAngleToDirection converts an IL-2 POV angle (e.g. "180") to a directional name
// (e.g. "D"). Unknown angles are returned unchanged.
func PovAngleToDirection(angle string) string {
	if direction, ok := povAngleToDirection[angle]; ok {
		return direction
	}
	return angle
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
