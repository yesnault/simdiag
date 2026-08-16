package target

import (
	"encoding/xml"
	"fmt"
	"os"
	"simdiag/common"
	"strconv"
	"strings"
)

// Profile represents the XML structure of a TARGET FCF profile
type Profile struct {
	XMLName     xml.Name   `xml:"FastEventsMapping"`
	Version     Version    `xml:"Version"`
	ProjectData Project    `xml:"ProjectData"`
	EventsList  EventsList `xml:"EventsList"`
	LayersList  LayersList `xml:"LayersList"`
}

// LayersList contains layer switch definitions
type LayersList struct {
	Layers []Layer `xml:",any"` // Layer1, Layer2, etc.
}

// Layer represents a layer switch definition
type Layer struct {
	XMLName  xml.Name `xml:""`
	HidEvent HidEvent `xml:"HidEvent"`
	Toggle   bool     `xml:"Toggle"`
}

// Version contains version information
type Version struct {
	ProgramVersion string `xml:"ProgramVersionNumber"`
	ProjectVersion string `xml:"ProjectVersionNumber"`
}

// Project contains project metadata
type Project struct {
	CharGenRate           int    `xml:"CharGenRate"`
	KeyboardLayout        int    `xml:"KeyboardLayout"`
	PulseEventTime        int    `xml:"PulseEventTime"`
	MouseSensitivity      int    `xml:"MouseSensitivity"`
	AdvancedConfiguration bool   `xml:"AdvancedConfiguration"`
	SelectedDevices       string `xml:"SelectedDevices"` // e.g., "1001 1002 "
}

// EventsList contains all events
type EventsList struct {
	Events []Event `xml:",any"`
}

// Event represents an event entry (Event0, Event1, etc.)
type Event struct {
	XMLName  xml.Name `xml:""`
	HidEvent HidEvent `xml:"HidEvent"`
}

// HidEvent represents an input event from a physical device
type HidEvent struct {
	DeviceNumber int         `xml:"DeviceNumber"`
	Name         string      `xml:"Name"`
	HidType      int         `xml:"HidType"` // 1=keyboard, 2=mouse, 3=joystick button, 4=axis?
	EventType    int         `xml:"EventType"`
	ActionType   int         `xml:"ActionType"`
	ControlIndex string      `xml:"ControlIndex"` // Can be int or string like "DXHATUPRIGHT"
	Events       HidCommands `xml:"Events"`
}

// HidCommands contains the list of output commands
type HidCommands struct {
	Commands []HidCommand `xml:",any"`
}

// HidCommand represents an output command (keyboard/mouse/joystick output)
type HidCommand struct {
	XMLName        xml.Name      `xml:""`
	EventName      string        `xml:"EventName"`
	IsSequence     bool          `xml:"IsSequence"`
	Layers         string        `xml:"Layers"` // e.g., "1 16 32 64" or "2 16 32 64"
	Delay          int           `xml:"Delay"`
	Comment        string        `xml:"Comment"`
	EventsNumber   int           `xml:"EventsNumber"`
	StartTrigger   int           `xml:"StartTrigger"`
	StopTrigger    int           `xml:"StopTrigger"`
	CurveType      int           `xml:"CurveType"`
	AxisIsReversed bool          `xml:"AxisIsReversed"`
	AxisIsRelative bool          `xml:"AxisIsRelative"`
	OutputEvents   []OutputEvent `xml:",any"`
}

// OutputEvent represents an output event (HidEvent0, HidEvent1, etc.)
type OutputEvent struct {
	XMLName      xml.Name `xml:""`
	DeviceNumber int      `xml:"DeviceNumber"` // -1 = keyboard/mouse
	Name         string   `xml:"Name"`
	HidType      int      `xml:"HidType"`   // 1=keyboard, 2=mouse, 3=joystick
	EventType    int      `xml:"EventType"` // 1=press, 2=release, 3=pulse, 4=hold
	ActionType   int      `xml:"ActionType"`
	ControlIndex string   `xml:"ControlIndex"` // Can be int or string
}

// Binding represents a mapping from physical input to output
type Binding struct {
	DeviceNumber   int              // Source device (1001, 1002, etc.)
	InputName      string           // Physical input name (TG1, S2, H2U, etc.)
	InputType      common.InputType // Button, Axis, Hat
	InputID        string           // Normalized input ID
	EventName      string           // Action description from TARGET
	Trigger        string           // "I" = on press, "O" = on release, "" = both (hold)
	Layers         []int            // Active layers (1, 2, etc.) - deprecated, use LayerInfo
	LayerInfo      LayerInfo        // Structured layer information (I/O and U/M/D)
	OutputKeys     []string         // Keyboard keys output (e.g., ["L_ALT", "SPC"])
	OutputMouse    string           // Mouse output if any
	OutputJoystick string           // Virtual joystick output if any

	// KeyboardLayout is the layout OutputKeys are written in, taken from the
	// profile the binding was read from. TARGET records the character the
	// author's keyboard produced, so an AZERTY user's script says "é" where a
	// QWERTY one says "2"; matching against a simulator needs to know which.
	KeyboardLayout KeyboardLayout
}

// ParseProfile parses a TARGET FCF profile file
func ParseProfile(profilePath string) ([]*Binding, error) {
	if profilePath == "" {
		return nil, nil
	}

	profile, err := loadAndParseXML(profilePath)
	if err != nil {
		return nil, err
	}

	bindings, err := parseEvents(profile.EventsList.Events)
	if err != nil {
		return nil, err
	}

	// The layout applies to the whole profile, but it is the keys that need it,
	// so every binding carries the one its file declared.
	layout := keyboardLayoutFromTarget(profile.ProjectData.KeyboardLayout)
	for _, binding := range bindings {
		binding.KeyboardLayout = layout
	}

	return bindings, nil
}

// keyboardLayoutFromTarget maps the <KeyboardLayout> of a .fcf onto a layout.
//
// Thrustmaster does not document the encoding. 1 is AZERTY, established from real
// profiles: a file declaring 1 contains "é", "²" and "&", which only resolve
// against a simulator once treated as AZERTY. Other values do occur in the wild,
// so anything unrecognised stays QWERTY, the behaviour every profile got before
// the layout was read at all, which means no existing profile can regress.
func keyboardLayoutFromTarget(layout int) KeyboardLayout {
	if layout == targetLayoutAzerty {
		return KeyboardAZERTY
	}
	return KeyboardQWERTY
}

// targetLayoutAzerty is the only <KeyboardLayout> value whose meaning is known.
const targetLayoutAzerty = 1

// loadAndParseXML loads and parses the TARGET XML file
func loadAndParseXML(profilePath string) (*Profile, error) {
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("TARGET profile not found: %s", profilePath)
	}

	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading TARGET profile: %w", err)
	}

	var profile Profile
	if err := xml.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("error parsing TARGET XML: %w", err)
	}

	return &profile, nil
}

// parseEvents parses all events from the profile
func parseEvents(events []Event) ([]*Binding, error) {
	bindings := make([]*Binding, 0)

	for _, event := range events {
		eventBindings := parseSingleEvent(event)
		bindings = append(bindings, eventBindings...)
	}

	return bindings, nil
}

// parseSingleEvent parses a single event and returns its bindings
func parseSingleEvent(event Event) []*Binding {
	hidEvent := event.HidEvent
	inputType, inputID := parseTargetInputType(hidEvent.Name, hidEvent.ControlIndex)

	commandsByEvent, commandOrder := groupCommandsByEventName(hidEvent.Events.Commands)

	bindings := make([]*Binding, 0)
	for _, eventName := range commandOrder {
		commands := commandsByEvent[eventName]
		eventBindings := parseEventCommands(hidEvent, inputType, inputID, eventName, commands)
		bindings = append(bindings, eventBindings...)
	}

	return bindings
}

// groupCommandsByEventName groups commands by their event name
func groupCommandsByEventName(commands []HidCommand) (map[string][]*HidCommand, []string) {
	commandsByEvent := make(map[string][]*HidCommand)
	commandOrder := make([]string, 0)

	for i := range commands {
		cmd := &commands[i]
		if cmd.EventName == "" {
			continue
		}
		if _, exists := commandsByEvent[cmd.EventName]; !exists {
			commandOrder = append(commandOrder, cmd.EventName)
		}
		commandsByEvent[cmd.EventName] = append(commandsByEvent[cmd.EventName], cmd)
	}

	return commandsByEvent, commandOrder
}

// parseEventCommands parses commands for a single event
func parseEventCommands(hidEvent HidEvent, inputType common.InputType, inputID, eventName string, commands []*HidCommand) []*Binding {
	outputs := collectOutputs(commands)
	return createBindings(hidEvent, inputType, inputID, eventName, outputs)
}

// eventOutputs holds collected outputs for press and release
type eventOutputs struct {
	pressKeys             []string
	releaseKeys           []string
	pressMouseOutput      string
	releaseMouseOutput    string
	pressJoystickOutput   string
	releaseJoystickOutput string
	layers                []int
	layerInfo             LayerInfo
	hasHold               bool
	hasPulse              bool
}

// collectOutputs collects all outputs from commands
func collectOutputs(commands []*HidCommand) *eventOutputs {
	outputs := &eventOutputs{
		pressKeys:   make([]string, 0),
		releaseKeys: make([]string, 0),
	}

	pressKeysMap := make(map[string]bool)
	releaseKeysMap := make(map[string]bool)
	layerInfoParsed := false

	for _, cmd := range commands {
		if outputs.layers == nil {
			outputs.layers = parseTargetLayers(cmd.Layers)
		}
		if !layerInfoParsed {
			outputs.layerInfo = parseLayerInfo(cmd.Layers)
			layerInfoParsed = true
		}

		for _, outputEvent := range cmd.OutputEvents {
			if !strings.HasPrefix(outputEvent.XMLName.Local, "HidEvent") {
				continue
			}

			processOutputEvent(&outputEvent, outputs, pressKeysMap, releaseKeysMap)
		}
	}

	return outputs
}

// processOutputEvent processes a single output event
func processOutputEvent(outputEvent *OutputEvent, outputs *eventOutputs, pressKeysMap, releaseKeysMap map[string]bool) {
	isPress := outputEvent.EventType == 1 || outputEvent.EventType == 3 || outputEvent.EventType == 4
	isRelease := outputEvent.EventType == 2

	if outputEvent.EventType == 4 {
		outputs.hasHold = true
	}
	if outputEvent.EventType == 3 {
		outputs.hasPulse = true
	}

	switch outputEvent.HidType {
	case 1: // Keyboard
		processKeyboardOutput(outputEvent, outputs, pressKeysMap, releaseKeysMap, isPress, isRelease)
	case 2: // Mouse
		processMouseOutput(outputEvent, outputs, isPress, isRelease)
	case 3: // Joystick button
		processJoystickOutput(outputEvent, outputs, isPress, isRelease)
	}
}

// processKeyboardOutput processes keyboard output
func processKeyboardOutput(outputEvent *OutputEvent, outputs *eventOutputs, pressKeysMap, releaseKeysMap map[string]bool, isPress, isRelease bool) {
	keyName := targetKeyNameToStandard(outputEvent.Name)
	if keyName == "" {
		return
	}

	if isPress && !pressKeysMap[keyName] {
		pressKeysMap[keyName] = true
		outputs.pressKeys = append(outputs.pressKeys, keyName)
	}
	if isRelease && !releaseKeysMap[keyName] {
		releaseKeysMap[keyName] = true
		outputs.releaseKeys = append(outputs.releaseKeys, keyName)
	}
}

// processMouseOutput processes mouse output
func processMouseOutput(outputEvent *OutputEvent, outputs *eventOutputs, isPress, isRelease bool) {
	mouseName := fmt.Sprintf("Mouse %s", outputEvent.Name)
	if isPress && outputs.pressMouseOutput == "" {
		outputs.pressMouseOutput = mouseName
	}
	if isRelease && outputs.releaseMouseOutput == "" {
		outputs.releaseMouseOutput = mouseName
	}
}

// processJoystickOutput processes joystick output
func processJoystickOutput(outputEvent *OutputEvent, outputs *eventOutputs, isPress, isRelease bool) {
	if outputEvent.DeviceNumber != -1 {
		return
	}

	joystickName := fmt.Sprintf("vJoy BTN%s", outputEvent.ControlIndex)
	if isPress && outputs.pressJoystickOutput == "" {
		outputs.pressJoystickOutput = joystickName
	}
	if isRelease && outputs.releaseJoystickOutput == "" {
		outputs.releaseJoystickOutput = joystickName
	}
}

// createBindings creates bindings from collected outputs
func createBindings(hidEvent HidEvent, inputType common.InputType, inputID, eventName string, outputs *eventOutputs) []*Binding {
	hasPress := len(outputs.pressKeys) > 0 || outputs.pressMouseOutput != "" || outputs.pressJoystickOutput != ""
	hasRelease := len(outputs.releaseKeys) > 0 || outputs.releaseMouseOutput != "" || outputs.releaseJoystickOutput != ""

	sameOutputs := outputs.hasHold || outputs.hasPulse || (hasPress && hasRelease && keysEqual(outputs.pressKeys, outputs.releaseKeys))

	switch {
	case sameOutputs || (hasPress && !hasRelease):
		return createHoldOrPressBinding(hidEvent, inputType, inputID, eventName, outputs, hasPress)
	case hasPress && hasRelease && !sameOutputs:
		return createPressReleaseBindings(hidEvent, inputType, inputID, eventName, outputs)
	case !hasPress && hasRelease:
		return createReleaseOnlyBinding(hidEvent, inputType, inputID, eventName, outputs)
	default:
		return nil
	}
}

// createHoldOrPressBinding creates a binding for hold or press-only actions
func createHoldOrPressBinding(hidEvent HidEvent, inputType common.InputType, inputID, eventName string, outputs *eventOutputs, hasPress bool) []*Binding {
	if !hasPress {
		return nil
	}

	binding := &Binding{
		DeviceNumber:   hidEvent.DeviceNumber,
		InputName:      hidEvent.Name,
		InputType:      inputType,
		InputID:        inputID,
		EventName:      eventName,
		Trigger:        "",
		Layers:         outputs.layers,
		LayerInfo:      outputs.layerInfo,
		OutputKeys:     outputs.pressKeys,
		OutputMouse:    outputs.pressMouseOutput,
		OutputJoystick: outputs.pressJoystickOutput,
	}

	return []*Binding{binding}
}

// createPressReleaseBindings creates separate bindings for press and release
func createPressReleaseBindings(hidEvent HidEvent, inputType common.InputType, inputID, eventName string, outputs *eventOutputs) []*Binding {
	pressBinding := &Binding{
		DeviceNumber:   hidEvent.DeviceNumber,
		InputName:      hidEvent.Name,
		InputType:      inputType,
		InputID:        inputID,
		EventName:      eventName,
		Trigger:        "",
		Layers:         outputs.layers,
		LayerInfo:      outputs.layerInfo,
		OutputKeys:     outputs.pressKeys,
		OutputMouse:    outputs.pressMouseOutput,
		OutputJoystick: outputs.pressJoystickOutput,
	}

	releaseBinding := &Binding{
		DeviceNumber:   hidEvent.DeviceNumber,
		InputName:      hidEvent.Name,
		InputType:      inputType,
		InputID:        inputID,
		EventName:      eventName,
		Trigger:        "",
		Layers:         outputs.layers,
		LayerInfo:      outputs.layerInfo,
		OutputKeys:     outputs.releaseKeys,
		OutputMouse:    outputs.releaseMouseOutput,
		OutputJoystick: outputs.releaseJoystickOutput,
	}

	return []*Binding{pressBinding, releaseBinding}
}

// createReleaseOnlyBinding creates a binding for release-only actions
func createReleaseOnlyBinding(hidEvent HidEvent, inputType common.InputType, inputID, eventName string, outputs *eventOutputs) []*Binding {
	binding := &Binding{
		DeviceNumber:   hidEvent.DeviceNumber,
		InputName:      hidEvent.Name,
		InputType:      inputType,
		InputID:        inputID,
		EventName:      eventName,
		Trigger:        "",
		Layers:         outputs.layers,
		LayerInfo:      outputs.layerInfo,
		OutputKeys:     outputs.releaseKeys,
		OutputMouse:    outputs.releaseMouseOutput,
		OutputJoystick: outputs.releaseJoystickOutput,
	}

	return []*Binding{binding}
}

// keysEqual checks if two slices of keys contain the same elements (order independent)
func keysEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]bool)
	for _, k := range a {
		aMap[k] = true
	}
	for _, k := range b {
		if !aMap[k] {
			return false
		}
	}
	return true
}

// parseTargetInputType determines the input type from TARGET naming conventions
// targetAxisNames maps the common TARGET axis names to DirectInput axis IDs.
var targetAxisNames = map[string]string{
	"JOYX": "X", "JOYY": "Y", "JOYRZ": "RZ",
	"THR": "Z", "THROTTLE": "Z",
	"SCX": "RX", "SCY": "RY", // Slew control
	"MRX": "RX", "MRY": "RY", // Mini stick
	"SLIDER": "SLIDER",
}

// hatDirectionSuffix maps the trailing letter of a TARGET hat name (H1U, CSD, ...)
// to its direction. Returns "" for any other letter.
func hatDirectionSuffix(c byte) string {
	switch c {
	case 'U', 'D', 'L', 'R':
		return string(c)
	}
	return ""
}

// parseTargetHatName recognises the two TARGET spellings of a POV hat: the
// joystick's H1U/H1D/... and the throttle's Coolie Switch CSU/CSD/... Both are
// only real hats when the ControlIndex says DXHAT; otherwise they are buttons
// (H2, H3 and H4 on the Warthog).
func parseTargetHatName(name, controlIndexUpper string) (string, bool) {
	if !strings.HasPrefix(controlIndexUpper, "DXHAT") {
		return "", false
	}

	// Joystick hats: H<n><direction>
	if len(name) >= 2 && name[0] == 'H' && name[1] >= '1' && name[1] <= '9' {
		direction := ""
		if len(name) >= 3 {
			direction = hatDirectionSuffix(name[2])
		}
		return string(name[1]) + "_" + direction, true
	}

	// Warthog Throttle Coolie Switch: CS<direction>, always hat 1
	if len(name) == 3 && strings.HasPrefix(name, "CS") {
		if direction := hatDirectionSuffix(name[2]); direction != "" {
			return "1_" + direction, true
		}
	}

	return "", false
}

func parseTargetInputType(name string, controlIndex string) (common.InputType, string) {
	name = strings.ToUpper(name)
	controlIndexUpper := strings.ToUpper(controlIndex)

	if inputID, ok := parseTargetHatName(name, controlIndexUpper); ok {
		return common.Hat, inputID
	}

	if axisID, ok := targetAxisNames[name]; ok {
		return common.Axis, axisID
	}

	// The ControlIndex itself can name a hat direction (DXHATUP, DXHATUPRIGHT, ...)
	if hatDir, found := strings.CutPrefix(controlIndexUpper, "DXHAT"); found {
		return common.Hat, parseHatDirection(hatDir)
	}

	// Button detection - map TARGET physical name to standard button number
	// Try to map physical name first, fall back to ControlIndex
	if buttonID := mapTargetButtonName(name); buttonID != "" {
		return common.Button, buttonID
	}

	// Fall back to ControlIndex as button number
	return common.Button, controlIndex
}

// mapTargetButtonName maps TARGET physical button names to standard button numbers
// This is needed because TARGET assigns virtual DirectX button numbers that don't
// match the physical layout shown on device templates
func mapTargetButtonName(name string) string {
	// Warthog Throttle button mapping (standard DirectX numbers without TARGET)
	// Reference: https://forums.frontier.co.uk/threads/thrustmaster-warthog-configuration-sheets.22379/
	// Note: Non-native DX positions (BSM, FLAPM, APUOFF, EACOFF, RDRDIS) mapped to their physical switch companion
	throttleButtons := map[string]string{
		"SC":       "1",  // Slew Click
		"MSP":      "2",  // Ministick Push
		"MSU":      "3",  // Ministick Up
		"MSR":      "4",  // Ministick Right
		"MSD":      "5",  // Ministick Down
		"MSL":      "6",  // Ministick Left
		"SPDF":     "7",  // Speedbrake Forward
		"SPDB":     "8",  // Speedbrake Back
		"BSF":      "9",  // Boat Switch Forward
		"BSB":      "10", // Boat Switch Back
		"CHF":      "11", // China Hat Forward
		"CHB":      "12", // China Hat Back
		"PSF":      "13", // Pinky Switch Forward
		"PSB":      "14", // Pinky Switch Back
		"LTB":      "15", // Left Throttle Button
		"EFLNORM":  "16", // Engine Fuel Flow Left Normal
		"EFRNORM":  "17", // Engine Fuel Flow Right Normal
		"EOLMOTOR": "18", // Engine Oper Left Motor
		"EORMOTOR": "19", // Engine Oper Right Motor
		"APUON":    "20", // APU Start
		"LDGH":     "21", // Landing Gear Horn Silence
		"FLAPU":    "22", // Flaps Up
		"FLAPD":    "23", // Flaps Down
		"EACON":    "24", // EAC On
		"RDRNRM":   "25", // Radar Altimeter Normal
		"APENG":    "26", // Autopilot Engage/Disengage
		"APPAT":    "27", // Autopilot Path
		"APALT":    "28", // Autopilot Altitude
		// Legacy/alternate names (some TARGET scripts use these)
		"EFLOVER": "18", // Alias for EOLMOTOR
		"EFROVER": "19", // Alias for EORMOTOR
		// Non-native DX positions mapped to their physical switch companion
		"APUOFF": "20", // APU Off -> same switch as APUON
		"EACOFF": "24", // EAC Off -> same switch as EACON
		"RDRDIS": "25", // Radar Disable -> same switch as RDRNRM
		"BSM":    "9",  // Boat Switch Middle -> mapped to BSF (forward/up position)
		"FLAPM":  "22", // Flaps Middle -> mapped to FLAPU (up position)
	}

	// Warthog Joystick button mapping
	joystickButtons := map[string]string{
		"TG1": "1",  // Trigger First Stage
		"S2":  "2",  // Weapon Release
		"S3":  "3",  // NWS/MSL Step
		"S4":  "4",  // Paddle Switch
		"S1":  "5",  // Master Mode
		"TG2": "6",  // Trigger Second Stage
		"H2U": "7",  // TMS Up
		"H2R": "8",  // TMS Right
		"H2D": "9",  // TMS Down
		"H2L": "10", // TMS Left
		"H3U": "11", // DMS Up
		"H3R": "12", // DMS Right
		"H3D": "13", // DMS Down
		"H3L": "14", // DMS Left
		"H4U": "15", // CMS Up
		"H4R": "16", // CMS Right
		"H4D": "17", // CMS Down
		"H4L": "18", // CMS Left
		"H4P": "19", // CMS Push
	}

	name = strings.ToUpper(name)

	if id, ok := throttleButtons[name]; ok {
		return id
	}
	if id, ok := joystickButtons[name]; ok {
		return id
	}

	return ""
}

// IsMidPosition returns true if the button name represents a 3-position switch mid position
func IsMidPosition(name string) bool {
	midPositions := map[string]bool{
		"BSM":   true, // Boat Switch Middle
		"FLAPM": true, // Flaps Middle
		"PSM":   true, // Pinky Switch Middle
	}
	return midPositions[strings.ToUpper(name)]
}

// parseHatDirection converts TARGET hat direction names to standard format
func parseHatDirection(direction string) string {
	directionMap := map[string]string{
		"UP":        "1_U",
		"DOWN":      "1_D",
		"LEFT":      "1_L",
		"RIGHT":     "1_R",
		"UPRIGHT":   "1_UR",
		"UPLEFT":    "1_UL",
		"DOWNRIGHT": "1_DR",
		"DOWNLEFT":  "1_DL",
	}
	if mapped, ok := directionMap[direction]; ok {
		return mapped
	}
	return "1_" + direction
}

// LayerInfo contains parsed layer information from TARGET
// TARGET layer encoding:
// - 1 = I (shift/switch pressed)
// - 2 = O (shift/switch not pressed, default)
// - 16 = U (Up layer)
// - 32 = M (Medium layer, default)
// - 64 = D (Down layer)
type LayerInfo struct {
	HasI bool // Switch/shift is pressed (layer value 1)
	HasO bool // Switch/shift is NOT pressed (layer value 2)
	HasU bool // Up layer active (layer value 16)
	HasM bool // Medium layer active (layer value 32)
	HasD bool // Down layer active (layer value 64)
}

// HasSwitchActive returns true if the I switch is active (shift pressed)
// Layer 1 (HasI) = switch IS pressed (shift mode)
// Layer 2 (HasO) = switch NOT pressed (normal mode)
func (l LayerInfo) HasSwitchActive() bool {
	return l.HasI && !l.HasO
}

// parseTargetLayers parses the layers string into a list of layer numbers
// TARGET uses: "1 16 32 64" or "2 16 32 64"
// - 1 = I (shift pressed), 2 = O (shift not pressed)
// - 16 = U, 32 = M, 64 = D
// Returns old format for backward compatibility (will be deprecated)
func parseTargetLayers(layersStr string) []int {
	layers := make([]int, 0)
	parts := strings.Fields(layersStr)

	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		// Only keep I/O values (1-2), ignore U/M/D flags (16, 32, 64)
		// This maintains backward compatibility
		if num >= 1 && num <= 2 {
			layers = append(layers, num)
		}
	}

	return layers
}

// parseLayerInfo parses the layers string into structured layer information
func parseLayerInfo(layersStr string) LayerInfo {
	info := LayerInfo{}
	parts := strings.Fields(layersStr)

	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		switch num {
		case 1:
			info.HasI = true
		case 2:
			info.HasO = true
		case 16:
			info.HasU = true
		case 32:
			info.HasM = true
		case 64:
			info.HasD = true
		}
	}

	return info
}

// targetKeyNameToStandard converts TARGET key names to standard names
func targetKeyNameToStandard(name string) string {
	if name == "" {
		return ""
	}

	// Common TARGET key name mappings
	mappings := map[string]string{
		"L_ALT":   "LAlt",
		"R_ALT":   "RAlt",
		"L_SHIFT": "LShift",
		"R_SHIFT": "RShift",
		"L_CTRL":  "LCtrl",
		"R_CTRL":  "RCtrl",
		"L_CTL":   "LCtrl",
		"R_CTL":   "RCtrl",
		"L_WIN":   "LWin",
		"R_WIN":   "RWin",
		"SPC":     "Space",
		"ENT":     "Enter",
		"BSP":     "Backspace",
		"TAB":     "Tab",
		"ESC":     "ESC",
		"DEL":     "Delete",
		"INS":     "Insert",
		"HOME":    "Home",
		"END":     "End",
		"PGUP":    "PgUp",
		"PGDN":    "PgDn",
		"UP":      "Up",
		"DOWN":    "Down",
		"LEFT":    "Left",
		"RIGHT":   "Right",
		"UARROW":  "Up",
		"DARROW":  "Down",
		"LARROW":  "Left",
		"RARROW":  "Right",
		"CAPS":    "CapsLock",
		"NUMLOCK": "NumLock",
		"SCRLCK":  "ScrollLock",
		"PRTSC":   "PrintScreen",
		"PAUSE":   "Pause",
		// Symbol keys
		")":  ")",
		"(":  "(",
		"+":  "+",
		"-":  "-",
		"=":  "=",
		"[":  "[",
		"]":  "]",
		";":  ";",
		"'":  "'",
		",":  ",",
		".":  ".",
		"/":  "/",
		"\\": "\\",
		"`":  "`",
		"*":  "*",
		"&":  "&",
		"%":  "%",
		"$":  "$",
		"#":  "#",
		"@":  "@",
		"!":  "!",
		"~":  "~",
		"|":  "|",
		"<":  "<",
		">":  ">",
		"?":  "?",
		"{":  "{",
		"}":  "}",
		"_":  "_",
		"\"": "\"",
	}

	upperName := strings.ToUpper(name)
	if standard, ok := mappings[upperName]; ok {
		return standard
	}

	// Function keys (F1-F12)
	if len(name) >= 2 && (name[0] == 'F' || name[0] == 'f') {
		if _, err := strconv.Atoi(name[1:]); err == nil {
			return strings.ToUpper(name)
		}
	}

	// Numpad keys
	if strings.HasPrefix(upperName, "KP") || strings.HasPrefix(upperName, "NUM") {
		return "Num" + strings.TrimPrefix(strings.TrimPrefix(upperName, "KP"), "NUM")
	}

	// Single letter keys - return uppercase
	if len(name) == 1 {
		return strings.ToUpper(name)
	}

	return name
}

// LoadBindings loads TARGET bindings for a specific device number
func LoadBindings(profilePath string, deviceNumber int) []*Binding {
	if profilePath == "" {
		return nil
	}

	allBindings, err := ParseProfile(profilePath)
	if err != nil {
		common.Printf("⚠ Error parsing TARGET profile: %v\n", err)
		return nil
	}

	// Filter bindings for this device
	deviceBindings := make([]*Binding, 0)
	for _, binding := range allBindings {
		if binding.DeviceNumber == deviceNumber {
			deviceBindings = append(deviceBindings, binding)
		}
	}

	return deviceBindings
}

// GetTargetDeviceNumbers returns the list of device numbers used in the profile
func GetTargetDeviceNumbers(profilePath string) ([]int, error) {
	if profilePath == "" {
		return nil, nil
	}

	// Check if file exists
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("TARGET profile not found: %s", profilePath)
	}

	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading TARGET profile: %w", err)
	}

	var profile Profile
	if err := xml.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("error parsing TARGET XML: %w", err)
	}

	// Parse SelectedDevices string (e.g., "1001 1002 ")
	deviceNumbers := make([]int, 0)
	parts := strings.Fields(profile.ProjectData.SelectedDevices)
	for _, part := range parts {
		if num, err := strconv.Atoi(part); err == nil {
			deviceNumbers = append(deviceNumbers, num)
		}
	}

	return deviceNumbers, nil
}

// DeviceNumberToName returns a human-readable name for TARGET device numbers
// Common Thrustmaster device numbers:
// 1001 = Joystick (Warthog/T.16000M)
// 1002 = Throttle (Warthog TWCS/HOTAS)
// 1003 = MFD Cougar
// 1004 = Rudder Pedals
func DeviceNumberToName(deviceNumber int) string {
	names := map[int]string{
		1001: "Joystick",
		1002: "Throttle",
		1003: "MFD",
		1004: "Rudder Pedals",
		1005: "Device 5",
		1006: "Device 6",
	}

	if name, ok := names[deviceNumber]; ok {
		return name
	}
	return fmt.Sprintf("Device %d", deviceNumber)
}

// targetDevicePatterns maps TARGET device numbers to name patterns for auto-detection
// Each device number has a list of patterns that could match physical device names
var targetDevicePatterns = map[int][]string{
	1001: {"joystick", "stick", "grip"}, // Joystick (Warthog, T.16000M, etc.)
	1002: {"throttle", "twcs"},          // Throttle (Warthog Throttle, TWCS, etc.)
	1003: {"mfd", "cougar"},             // MFD Cougar
	1004: {"rudder", "pedal", "tfrp"},   // Rudder Pedals (TFRP, TPR, etc.)
}

// AutoMatchTargetDevices attempts to automatically match TARGET device numbers to physical devices
// Returns a slice of TargetDeviceMapping for successfully matched devices
func AutoMatchTargetDevices(targetDeviceNumbers []int, physicalDevices []*common.Device) []common.TargetDeviceMapping {
	mappings := make([]common.TargetDeviceMapping, 0)
	usedDevices := make(map[string]bool) // Track which physical devices have been matched

	for _, targetNum := range targetDeviceNumbers {
		patterns, hasPatterns := targetDevicePatterns[targetNum]
		if !hasPatterns {
			continue
		}

		// Try to find a matching physical device
		for _, device := range physicalDevices {
			if device == nil || device.IsVirtual {
				continue
			}

			// Skip already matched devices
			if usedDevices[device.GUID] {
				continue
			}

			deviceNameLower := strings.ToLower(device.Name)

			// Check if device name matches any pattern for this TARGET device
			for _, pattern := range patterns {
				if strings.Contains(deviceNameLower, pattern) {
					// Found a match!
					mapping := common.TargetDeviceMapping{
						DeviceNumber: targetNum,
						DeviceGUID:   device.GUID,
						DeviceName:   device.Name,
					}
					mappings = append(mappings, mapping)
					usedDevices[device.GUID] = true
					break
				}
			}

			// If we found a match for this TARGET device, move to next
			if usedDevices[device.GUID] {
				break
			}
		}
	}

	return mappings
}

// GetUnmatchedTargetDevices returns TARGET device numbers that were not auto-matched
func GetUnmatchedTargetDevices(targetDeviceNumbers []int, mappings []common.TargetDeviceMapping) []int {
	matched := make(map[int]bool)
	for _, m := range mappings {
		matched[m.DeviceNumber] = true
	}

	unmatched := make([]int, 0)
	for _, num := range targetDeviceNumbers {
		if !matched[num] {
			unmatched = append(unmatched, num)
		}
	}
	return unmatched
}

// LayerSwitcher contains information about a layer switcher button
type LayerSwitcher struct {
	DeviceNumber int
	ButtonID     string
}

// ParseLayerSwitchers extracts layer switcher information from a TARGET profile
// Returns a map of layer number -> switcher info (device + button)
func ParseLayerSwitchers(profilePath string) map[int]LayerSwitcher {
	layerSwitchers := make(map[int]LayerSwitcher)

	if profilePath == "" {
		return layerSwitchers
	}

	// Check if file exists
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return layerSwitchers
	}

	data, err := os.ReadFile(profilePath)
	if err != nil {
		return layerSwitchers
	}

	var profile Profile
	if err := xml.Unmarshal(data, &profile); err != nil {
		return layerSwitchers
	}

	// Parse LayersList
	for _, layer := range profile.LayersList.Layers {
		// Extract layer number from XML name (e.g., "Layer1" -> 1, but TARGET Layer1 = Layer 2 in code)
		layerName := layer.XMLName.Local
		if !strings.HasPrefix(layerName, "Layer") {
			continue
		}

		layerNumStr := strings.TrimPrefix(layerName, "Layer")
		layerNum, err := strconv.Atoi(layerNumStr)
		if err != nil {
			continue
		}

		// In TARGET .fcf, Layer1 in LayersList corresponds to Layer 2 in the bindings
		// So we add 1 to the layer number
		actualLayerNum := layerNum + 1

		// Parse the input type and ID
		inputType, inputID := parseTargetInputType(layer.HidEvent.Name, layer.HidEvent.ControlIndex)

		// Build the button ID
		var buttonID string
		switch inputType {
		case common.Button:
			buttonID = fmt.Sprintf("BTN%s", inputID)
		case common.Hat:
			buttonID = fmt.Sprintf("POV_%s", inputID)
		case common.Axis:
			buttonID = fmt.Sprintf("AXIS_%s", inputID)
		default:
			buttonID = layer.HidEvent.Name
		}

		layerSwitchers[actualLayerNum] = LayerSwitcher{
			DeviceNumber: layer.HidEvent.DeviceNumber,
			ButtonID:     buttonID,
		}
	}

	return layerSwitchers
}
