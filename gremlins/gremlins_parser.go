package gremlins

import (
	"encoding/xml"
	"fmt"
	"os"
	"simdiag/common"
	"strconv"
	"strings"
)

// Profile represents the XML structure of a Gremlins profile
type Profile struct {
	XMLName xml.Name `xml:"profile"`
	Version string   `xml:"version,attr"`
	Devices []Device `xml:"devices>device"`
}

// Device represents a device in Gremlins
type Device struct {
	DeviceGUID string `xml:"device-guid,attr"`
	Name       string `xml:"name,attr"`
	Type       string `xml:"type,attr"`
	Modes      []Mode `xml:"mode"`
}

// Mode represents a mode in Gremlins
type Mode struct {
	Name    string   `xml:"name,attr"`
	Axes    []Axis   `xml:"axis"`
	Buttons []Button `xml:"button"`
	Hats    []Hat    `xml:"hat"`
}

// Axis represents an axis configuration
type Axis struct {
	ID          string     `xml:"id,attr"`
	Description string     `xml:"description,attr"`
	Container   *Container `xml:"container"`
}

// Button represents a button configuration
type Button struct {
	ID          string      `xml:"id,attr"`
	Description string      `xml:"description,attr"`
	Containers  []Container `xml:"container"`
}

// Hat represents a hat configuration
type Hat struct {
	ID          string     `xml:"id,attr"`
	Description string     `xml:"description,attr"`
	Container   *Container `xml:"container"`
}

// Container represents an action container
type Container struct {
	ActionSet *ActionSet `xml:"action-set"`
}

// ActionSet represents a set of actions
type ActionSet struct {
	MapToKeyboard       *MapToKeyboard       `xml:"map-to-keyboard"`
	MapToMouse          *MapToMouse          `xml:"map-to-mouse"`
	Macro               *Macro               `xml:"macro"`
	Remap               *Remap               `xml:"remap"`
	TemporaryModeSwitch *TemporaryModeSwitch `xml:"temporary-mode-switch"`
}

// MapToKeyboard represents keyboard mapping
type MapToKeyboard struct {
	Keys []Key `xml:"key"`
}

// Key represents a keyboard key
type Key struct {
	ScanCode int  `xml:"scan-code,attr"`
	Extended bool `xml:"extended,attr"`
}

// MapToMouse represents mouse mapping
type MapToMouse struct {
	ButtonID    int  `xml:"button-id,attr"`
	Direction   int  `xml:"direction,attr"`
	MotionInput bool `xml:"motion-input,attr"`
}

// Macro represents a macro
type Macro struct {
	Actions MacroActions `xml:"actions"`
}

// MacroActions represents macro actions
type MacroActions struct {
	Mouse []MacroMouse `xml:"mouse"`
	Key   []MacroKey   `xml:"key"`
	VJoy  []MacroVJoy  `xml:"vjoy"`
}

// MacroVJoy represents a vJoy action in macro
type MacroVJoy struct {
	InputID   int    `xml:"input-id,attr"`
	InputType string `xml:"input-type,attr"` // "button", "axis", "hat"
	Value     string `xml:"value,attr"`      // "True" or "False"
	VJoyID    int    `xml:"vjoy-id,attr"`
}

// MacroMouse represents a mouse action in macro
type MacroMouse struct {
	Button string `xml:"button,attr"`
	Press  bool   `xml:"press,attr"`
}

// MacroKey represents a keyboard action in macro
type MacroKey struct {
	ScanCode int  `xml:"scan-code,attr"`
	Extended bool `xml:"extended,attr"`
	Press    bool `xml:"press,attr"`
}

// Remap represents a remap to vJoy
type Remap struct {
	Button int `xml:"button,attr"`
	Axis   int `xml:"axis,attr"`
	Hat    int `xml:"hat,attr"`
	VJoy   int `xml:"vjoy,attr"`
}

// TemporaryModeSwitch represents a temporary mode switch
// This is equivalent to a "Modifier" button in DCS World
type TemporaryModeSwitch struct {
	Name string `xml:"name,attr"`
}

// Binding represents a mapping from device input to keyboard/mouse
type Binding struct {
	DeviceGUID     string
	DeviceName     string
	InputType      common.InputType
	InputID        string
	KeyboardKey    string // e.g., "ESC", "F1", "A"
	MouseButton    string // e.g., "Left", "Right", "Button 11"
	VJoyDevice     int    // vJoy device number (1, 2, etc.)
	VJoyButton     int    // vJoy button number
	VJoyAxis       int    // vJoy axis number
	VJoyHat        int    // vJoy hat number
	Description    string // Custom description from Gremlins
	Mode           string // Mode name (Base, Shift, Shift1, etc.)
	IsModeSwitcher bool   // True if this button activates a mode (equivalent to DCS "Modifier")
	SwitchesTo     string // Mode name this button switches to (e.g., "Shift", "Shift1")
}

// ParseProfile parses a Gremlins XML profile
func ParseProfile(profilePath string) ([]*Binding, error) {
	if profilePath == "" {
		return nil, nil
	}

	// Check if file exists
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("gremlins profile not found: %s", profilePath)
	}

	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading gremlins profile: %w", err)
	}

	var profile Profile
	if err := xml.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("error parsing gremlins XML: %w", err)
	}

	bindings := make([]*Binding, 0)

	for _, device := range profile.Devices {
		// Normalize GUID to uppercase without braces (to match DCS format)
		deviceGUID := common.NormalizeGUID(device.DeviceGUID)

		for _, mode := range device.Modes {
			bindings = append(bindings, parseModeButtons(mode, device, deviceGUID)...)
			bindings = append(bindings, parseModeAxes(mode, device, deviceGUID)...)
			bindings = append(bindings, parseModeHats(mode, device, deviceGUID)...)
		}
	}

	return bindings, nil
}

// parseModeButtons converts the button entries of one mode into bindings. A button
// can carry several containers (pressed, released, ...), each producing a binding.
func parseModeButtons(mode Mode, device Device, deviceGUID string) []*Binding {
	var bindings []*Binding

	for _, btn := range mode.Buttons {
		for _, container := range btn.Containers {
			if container.ActionSet == nil {
				continue
			}

			binding := &Binding{
				DeviceGUID:  deviceGUID,
				DeviceName:  device.Name,
				InputType:   common.Button,
				InputID:     btn.ID,
				Description: btn.Description,
				Mode:        mode.Name,
			}

			if applyButtonAction(binding, container.ActionSet) {
				bindings = append(bindings, binding)
			}
		}
	}

	return bindings
}

// applyButtonAction fills in what a button container does. It reports false when
// the container holds no action this tool understands.
func applyButtonAction(binding *Binding, actionSet *ActionSet) bool {
	// Temporary mode switch takes priority over any remap
	if tempModeSwitch := actionSet.TemporaryModeSwitch; tempModeSwitch != nil {
		binding.IsModeSwitcher = true
		binding.SwitchesTo = tempModeSwitch.Name
		return true
	}

	if remap := actionSet.Remap; remap != nil {
		binding.VJoyDevice = remap.VJoy
		binding.VJoyButton = remap.Button
		binding.VJoyAxis = remap.Axis
		return true
	}

	if mapKb := actionSet.MapToKeyboard; mapKb != nil && len(mapKb.Keys) > 0 {
		key := mapKb.Keys[0]
		binding.KeyboardKey = scanCodeToKeyName(key.ScanCode, key.Extended)
		return true
	}

	if mapMouse := actionSet.MapToMouse; mapMouse != nil {
		binding.MouseButton = fmt.Sprintf("Mouse %d", mapMouse.ButtonID)
		return true
	}

	if macro := actionSet.Macro; macro != nil {
		return applyMacroAction(binding, macro)
	}

	return false
}

// applyMacroAction fills in a binding from the first usable action of a macro:
// a mouse button, a key combination, or a vJoy button press.
func applyMacroAction(binding *Binding, macro *Macro) bool {
	if len(macro.Actions.Mouse) > 0 {
		binding.MouseButton = fmt.Sprintf("Mouse %s", macro.Actions.Mouse[0].Button)
		return true
	}

	if len(macro.Actions.Key) > 0 {
		// Build the key combination from the presses only, ignoring releases
		var keys []string
		for _, key := range macro.Actions.Key {
			if key.Press {
				keys = append(keys, scanCodeToKeyName(key.ScanCode, key.Extended))
			}
		}
		if len(keys) > 0 {
			binding.KeyboardKey = strings.Join(keys, " + ")
			return true
		}
	}

	// Macros can also press a vJoy button directly
	for _, vjoyAction := range macro.Actions.VJoy {
		if vjoyAction.InputType == "button" && vjoyAction.Value == "True" {
			binding.VJoyDevice = vjoyAction.VJoyID
			binding.VJoyButton = vjoyAction.InputID
			return true
		}
	}

	return false
}

// parseModeAxes converts the axis entries of one mode into bindings.
func parseModeAxes(mode Mode, device Device, deviceGUID string) []*Binding {
	var bindings []*Binding

	for _, axis := range mode.Axes {
		if axis.Container == nil || axis.Container.ActionSet == nil {
			continue
		}

		binding := &Binding{
			DeviceGUID:  deviceGUID,
			DeviceName:  device.Name,
			InputType:   common.Axis,
			InputID:     convertAxisIDToName(axis.ID),
			Description: axis.Description,
			Mode:        mode.Name,
		}

		switch {
		case axis.Container.ActionSet.Remap != nil:
			remap := axis.Container.ActionSet.Remap
			binding.VJoyDevice = remap.VJoy
			binding.VJoyAxis = remap.Axis
			bindings = append(bindings, binding)

		case axis.Container.ActionSet.MapToMouse != nil:
			// Mouse movement is a common axis target
			direction := "Movement"
			switch axis.Container.ActionSet.MapToMouse.Direction {
			case 0:
				direction = "Movement X"
			case 90:
				direction = "Movement Y"
			}
			binding.MouseButton = "Mouse " + direction
			bindings = append(bindings, binding)
		}
	}

	return bindings
}

// parseModeHats converts the hat entries of one mode into bindings. A hat remapped
// onto vJoy yields one binding per direction, matching how IL-2 and the SVG
// templates address hats.
func parseModeHats(mode Mode, device Device, deviceGUID string) []*Binding {
	hatDirections := []string{"U", "D", "L", "R"}

	var bindings []*Binding

	for _, hat := range mode.Hats {
		if hat.Container == nil || hat.Container.ActionSet == nil {
			continue
		}

		newHatBinding := func(inputID string) *Binding {
			return &Binding{
				DeviceGUID:  deviceGUID,
				DeviceName:  device.Name,
				InputType:   common.Hat,
				InputID:     inputID,
				Description: hat.Description,
				Mode:        mode.Name,
			}
		}

		if remap := hat.Container.ActionSet.Remap; remap != nil {
			for _, dir := range hatDirections {
				binding := newHatBinding(hat.ID + "_" + dir) // e.g. "1_U"
				binding.VJoyDevice = remap.VJoy
				binding.VJoyHat = remap.Hat
				bindings = append(bindings, binding)
			}
			continue
		}

		if mapKb := hat.Container.ActionSet.MapToKeyboard; mapKb != nil && len(mapKb.Keys) > 0 {
			key := mapKb.Keys[0]
			binding := newHatBinding(hat.ID)
			binding.KeyboardKey = scanCodeToKeyName(key.ScanCode, key.Extended)
			bindings = append(bindings, binding)
		}
	}

	return bindings
}

// scanCodeToKeyName converts Windows scan code to readable key name
func scanCodeToKeyName(scanCode int, extended bool) string {
	// Common scan codes mapping
	scanCodes := map[int]string{
		1:  "ESC",
		2:  "1",
		3:  "2",
		4:  "3",
		5:  "4",
		6:  "5",
		7:  "6",
		8:  "7",
		9:  "8",
		10: "9",
		11: "0",
		12: "Minus",
		13: "Equals",
		14: "Backspace",
		15: "Tab",
		16: "Q",
		17: "W",
		18: "E",
		19: "R",
		20: "T",
		21: "Y",
		22: "U",
		23: "I",
		24: "O",
		25: "P",
		26: "LBracket",
		27: "RBracket",
		28: "Enter",
		29: "LCtrl",
		30: "A",
		31: "S",
		32: "D",
		33: "F",
		34: "G",
		35: "H",
		36: "J",
		37: "K",
		38: "L",
		39: "Semicolon",
		40: "Quote",
		41: "Grave",
		42: "LShift",
		43: "Backslash",
		44: "Z",
		45: "X",
		46: "C",
		47: "V",
		48: "B",
		49: "N",
		50: "M",
		51: "Comma",
		52: "Period",
		53: "Slash",
		54: "RShift",
		55: "Num*",
		56: "LAlt",
		57: "Space",
		58: "CapsLock",
		59: "F1",
		60: "F2",
		61: "F3",
		62: "F4",
		63: "F5",
		64: "F6",
		65: "F7",
		66: "F8",
		67: "F9",
		68: "F10",
		69: "NumLock",
		70: "ScrollLock",
		71: "Num7",
		72: "Num8",
		73: "Num9",
		74: "Num-",
		75: "Num4",
		76: "Num5",
		77: "Num6",
		78: "Num+",
		79: "Num1",
		80: "Num2",
		81: "Num3",
		82: "Num0",
		83: "Num.",
		87: "F11",
		88: "F12",
	}

	// Extended scan codes (when extended flag is true)
	if extended {
		extendedCodes := map[int]string{
			28: "NumEnter",
			29: "RCtrl",
			53: "Num/",
			56: "RAlt",
			71: "Home",
			72: "Up",
			73: "PgUp",
			75: "Left",
			77: "Right",
			79: "End",
			80: "Down",
			81: "PgDn",
			82: "Insert",
			83: "Delete",
			91: "LWin",
			92: "RWin",
		}
		if key, ok := extendedCodes[scanCode]; ok {
			return key
		}
	}

	if key, ok := scanCodes[scanCode]; ok {
		return key
	}

	return fmt.Sprintf("Key%d", scanCode)
}

// LoadBindings loads Gremlins bindings for a specific device
func LoadBindings(profilePath, deviceGUID string) []*Binding {
	if profilePath == "" {
		return nil
	}

	allBindings, err := ParseProfile(profilePath)
	if err != nil {
		common.Printf("⚠ Error parsing Gremlins profile: %v\n", err)
		return nil
	}

	// Filter bindings for this device
	// Normalize both GUIDs for comparison (without braces to match DCS format)
	normalizedDeviceGUID := common.NormalizeGUID(deviceGUID)

	deviceBindings := make([]*Binding, 0)
	for _, binding := range allBindings {
		if common.MatchGUIDPartial(binding.DeviceGUID, normalizedDeviceGUID) {
			deviceBindings = append(deviceBindings, binding)
		}
	}
	return deviceBindings
}

// axisNames maps a Gremlins axis number to the standard name DCS and IL-2 use.
//
// Axes 4 and 5 are swapped relative to the obvious order; that is what WINWING
// hardware reports and it is deliberate. The matcher in gremlins.go used to
// carry its own copy of this table with 4 and 5 the other way round, so the
// parser and the matcher of the same package disagreed about which axis a
// binding was on.
var axisNames = map[int]string{
	1: "X",
	2: "Y",
	3: "Z",
	4: "RY", // WINWING: axis 4 = RY (mouse Y movement)
	5: "RX", // WINWING: axis 5 = RX (inverted from standard)
	6: "RZ",
	7: "SLIDER_1",
	8: "SLIDER_2",
}

// AxisName returns the standard name of a Gremlins axis number, and whether the
// number is one this mapping knows.
func AxisName(axisNumber int) (string, bool) {
	name, found := axisNames[axisNumber]
	return name, found
}

// convertAxisIDToName converts Gremlins numeric axis ID to standard axis name
// Used to match DCS axis naming (X, Y, Z, RX, RY, RZ, SLIDER_1, SLIDER_2)
func convertAxisIDToName(axisID string) string {
	number, err := strconv.Atoi(axisID)
	if err != nil {
		return axisID
	}
	if name, found := AxisName(number); found {
		return name
	}
	// Return as-is if not in map (for any custom axes)
	return axisID
}

// GetProfilePath returns the Gremlins profile path configured for a simulator.
//
// It takes no module: a pilot runs one Gremlins profile for the whole game, and
// the per-module paths that used to override this were only ever set to the same
// value repeated once per aircraft.
func GetProfilePath(config *common.Config, simType common.SimulationType) string {
	return config.GremlinsProfilePath(simType)
}
