package common

import "strings"

// SimdiagVersion holds the version string set by the main package
var SimdiagVersion = "dev"

// SimulatorParser defines the interface for parsing simulator configurations
type SimulatorParser interface {
	// Parse reads simulator configuration files and returns a ProfileCollection
	Parse(configPath string) (*ProfileCollection, error)
	// GetName returns the human-readable name of the simulator
	GetName() string
}

// BindingEnricher defines the interface for enriching bindings from external tools
type BindingEnricher interface {
	// Enrich adds bindings from external tools to an ExportDevice
	Enrich(exportDevice *ExportDevice, fullProfile *Profile, config *Config)
}

// SimulationType represents the simulation type
type SimulationType string

const (
	DCSWorld SimulationType = "DCS World"
	// IL2Sturmovik is IL-2 Sturmovik Great Battles (text-based config format)
	IL2Sturmovik SimulationType = "IL-2 Sturmovik"
	// IL2Korea is IL-2 Sturmovik Korea (JSON-based config format)
	IL2Korea SimulationType = "IL-2 Korea"
)

// GetConfigKey returns the configuration key for the simulator type
func (s SimulationType) GetConfigKey() string {
	switch s {
	case DCSWorld:
		return "dcs_world"
	case IL2Sturmovik:
		return "il2_sturmovik"
	case IL2Korea:
		return "il2_korea"
	default:
		return string(s)
	}
}

// SimulationTypeForConfigKey is the inverse of GetConfigKey, for callers that
// iterate Config.Simulators and hold a key rather than a type. An unknown key
// yields an empty type, which SimulatorIsConfigured treats as non-DCS.
func SimulationTypeForConfigKey(key string) SimulationType {
	for _, simType := range []SimulationType{DCSWorld, IL2Sturmovik, IL2Korea} {
		if simType.GetConfigKey() == key {
			return simType
		}
	}
	return ""
}

// ModuleKey returns the value written to the CSV "Module" column: the DCS module
// name, or a fixed label for the non-modular simulators.
func ModuleKey(simType SimulationType, module string) string {
	switch simType {
	case DCSWorld:
		return module
	case IL2Sturmovik:
		return "il2"
	case IL2Korea:
		return "il2-korea"
	}
	return ""
}

// OutputSubdir returns the directory, under the configured output directory, that
// holds the diagrams for a simulator/module pair. An empty result means the
// diagrams go straight into the output directory.
func OutputSubdir(simType SimulationType, module string) string {
	switch {
	case simType == DCSWorld && module != "":
		return "dcs-" + NormalizeModuleName(module)
	case simType == IL2Sturmovik:
		return "il2"
	case simType == IL2Korea:
		return "il2-korea"
	}
	return ""
}

// ExportTitle returns the title stamped on a diagram. fallback is used for
// simulator/module pairs that have no fixed title (typically the device name).
func ExportTitle(simType SimulationType, module, fallback string) string {
	switch {
	case simType == DCSWorld && module != "":
		return "DCS World / " + strings.ToUpper(module)
	case simType == IL2Sturmovik:
		return "IL-2 Sturmovik"
	case simType == IL2Korea:
		return "IL-2 Korea"
	}
	return fallback
}

// Device represents a control device
type Device struct {
	GUID      string
	Name      string
	IsVirtual bool // True if this is a virtual device (vJoy, Thrustmaster Combined, etc.)
}

// InputType represents the input type (button, axis, hat)
type InputType string

const (
	Button InputType = "button"
	Axis   InputType = "axis"
	Hat    InputType = "hat"
)

// Modifier represents a modifier key (shift, ctrl, alt, etc.)
type Modifier struct {
	Keys       []string // Modifier keys (e.g. ["JOY_BTN1", "JOY_BTN2"])
	Action     string   // Action associated with this modifier
	IsSwitch   bool     // True if the modifier is a switch (Touche-) instead of a regular modifier
	DeviceName string   // Device name where the modifier button is located (e.g., "Joystick - HOTAS Warthog")
}

// Binding represents a link between a physical control and a game action
type Binding struct {
	DeviceGUID    string
	DeviceName    string
	InputType     InputType
	InputID       string // e.g. "1" for button 1, "X" for X axis, "U" for hat up
	Action        string
	Description   string
	Modifiers     []Modifier // List of modifiers for this binding
	ModifierNum   int        // Sequential modifier number (from CSV "Modifier Num" column)
	ModifierKey   string     // If this binding IS a modifier definition, this is the key (e.g., "JOY_BTN105", "GREMLINS_MODE_Shift")
	VirtualDevice string     // Virtual device name (e.g., "vJoy Device #1") if using Gremlins remap, empty otherwise
	VirtualInput  string     // Virtual input (e.g., "BTN27", "LShift + Quote") if using Gremlins, empty otherwise
}

// DisplayText returns the text shown for this binding in CSV exports and diagrams.
// IL-2 carries a human-readable Description (e.g. "Suralimentation"); DCS and SRS
// only have an Action.
func (b Binding) DisplayText() string {
	if b.Description != "" {
		return b.Description
	}
	return b.Action
}

// Profile represents a complete configuration profile
type Profile struct {
	Name              string
	SimType           SimulationType
	Module            string // For DCS: M-2000C, FA-18C_hornet, etc. Empty for Default/UiLayer/CommandMenu
	Devices           map[string]*Device
	Bindings          []Binding
	ModifierDeviceMap map[string]ModifierInfo // Map from JOY_BTN key to device info (from modifiers.lua)
}

// ModifierInfo stores information about a modifier/switch from modifiers.lua
type ModifierInfo struct {
	DeviceGUID string
	DeviceName string
	Key        string
	IsSwitch   bool
}

// ProfileCollection contains all found profiles
type ProfileCollection struct {
	Profiles []*Profile
}

// Template represents an SVG template file
type Template struct {
	FilePath  string
	Name      string
	RawData   string
	Buttons   []string // E.g. ["BUTTON_1", "BUTTON_2"]
	Axes      []string // E.g. ["AXIS_X", "AXIS_Y"]
	Hats      []string // E.g. ["POV_1_U", "POV_1_D"]
	Modifiers []string // E.g. ["BUTTON_1_Modifier_1"]
}

// TargetDeviceMapping maps TARGET device numbers to physical device GUIDs
type TargetDeviceMapping struct {
	DeviceNumber int    `yaml:"device_number"` // TARGET device number (1001, 1002, etc.)
	DeviceGUID   string `yaml:"device_guid"`   // Physical device GUID
	DeviceName   string `yaml:"device_name"`   // Device name for display
}

// ExportDevice groups a device, template and profile for export
type ExportDevice struct {
	Device          *Device
	Template        *Template
	Profile         *Profile
	OutputDirectory string
	SimulatorName   string
	SimdiagVersion  string
	Title           string
}

// ValidationError represents a binding that has no corresponding key in template
type ValidationError struct {
	DeviceName string         `json:"deviceName"`
	SimType    SimulationType `json:"simulator"`
	InputType  InputType      `json:"inputType"`
	InputID    string         `json:"inputId"`
	Action     string         `json:"action"`
	MissingKey string         `json:"missingKey"`
}
