package common

// SimdiagVersion holds the version string set by the main package
var SimdiagVersion = "dev"

// ExternalFuncs holds function dependencies that are injected from main
type ExternalFuncs struct {
	GetTargetDeviceNumbers             func(string) ([]int, error)
	AutoMatchTargetDevices             func([]int, []*Device) []TargetDeviceMapping
	TargetDeviceNumberToName           func(int) string
	GetUnmatchedTargetDevices          func([]int, []TargetDeviceMapping) []int
	LoadGremlinsBindingsForDevice      func(string, *Config) interface{} // Returns package-specific binding type
	LoadOpenKneeboardBindingsForDevice func(string, *Config) interface{} // Returns package-specific binding type
	ParseGremlinsProfile               func(string) (interface{}, error) // Returns package-specific binding type
}

// ExtFuncs is the global instance of external functions
var ExtFuncs *ExternalFuncs

// SimulatorParser defines the interface for parsing simulator configurations
type SimulatorParser interface {
	// Parse reads simulator configuration files and returns a ProfileCollection
	Parse(configPath string) (*ProfileCollection, error)
	// GetName returns the human-readable name of the simulator
	GetName() string
	// GetType returns the SimulationType
	GetType() SimulationType
}

// BindingEnricher defines the interface for enriching bindings from external tools
type BindingEnricher interface {
	// Enrich adds bindings from external tools to an ExportDevice
	Enrich(exportDevice *ExportDevice, fullProfile *Profile, config *Config)
	// GetName returns the human-readable name of the enricher
	GetName() string
	// IsAvailable checks if the enricher's configuration is available
	IsAvailable(config *Config) bool
}

// Configurable defines the interface for components that need interactive configuration
type Configurable interface {
	// Configure prompts the user for configuration and saves to config
	Configure(config *Config, batchMode bool) error
	// GetName returns the human-readable name of the configurable component
	GetName() string
}

// SimulationType represents the simulation type
type SimulationType string

const (
	DCSWorld     SimulationType = "DCS World"
	IL2Sturmovik SimulationType = "IL-2 Sturmovik"
)

// GetConfigKey returns the configuration key for the simulator type
func (s SimulationType) GetConfigKey() string {
	switch s {
	case DCSWorld:
		return "dcs_world"
	case IL2Sturmovik:
		return "il2_sturmovik"
	default:
		return string(s)
	}
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
	DeviceName string
	SimType    SimulationType
	InputType  InputType
	InputID    string
	Action     string
	MissingKey string
}
