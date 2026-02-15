package csv

import "strings"

// CSV column names (constants for maintainability)
const (
	ColSimulator          = "Simulator"
	ColModule             = "Module"
	ColAction             = "Action"
	ColModifier           = "Modifier"
	ColModifierDevice     = "Modifier Device"
	ColModifierNum        = "Modifier Num"
	ColPhysicalDevice     = "Physical Device"
	ColPhysicalInput      = "Physical Input"
	ColPhysicalDeviceGUID = "Physical Device GUID"
	ColVirtualDevice      = "Virtual Device"
	ColVirtualInput       = "Virtual Input"
	ColTemplateKey        = "Template Key"
	ColTemplate           = "Template"
)

// AllColumns returns all column names in order
var AllColumns = []string{
	ColSimulator,
	ColModule,
	ColAction,
	ColModifier,
	ColModifierDevice,
	ColModifierNum,
	ColPhysicalDevice,
	ColPhysicalInput,
	ColPhysicalDeviceGUID,
	ColVirtualDevice,
	ColVirtualInput,
	ColTemplateKey,
	ColTemplate,
}

// GetHeaderString returns the CSV header as a string
func GetHeaderString() string {
	return strings.Join(AllColumns, ",") + "\n"
}
