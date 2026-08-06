package common

import "strings"

// Prefixes used in the "Physical Input" / "Virtual Input" CSV columns.
const (
	buttonInputPrefix = "BTN"
	axisInputPrefix   = "Axis "
	hatInputPrefix    = "POV "
)

// FormatInputText renders an input as it appears in the CSV input columns:
// "BTN1", "Axis X", "POV 1_U". It is the inverse of ParseInputText.
func FormatInputText(inputType InputType, inputID string) string {
	switch inputType {
	case Button:
		return buttonInputPrefix + inputID
	case Axis:
		return axisInputPrefix + inputID
	case Hat:
		return hatInputPrefix + inputID
	}
	return ""
}

// ParseInputText parses the CSV representation produced by FormatInputText back
// into an input type and ID. The final return value reports whether the text was
// recognised.
func ParseInputText(text string) (InputType, string, bool) {
	switch {
	case strings.HasPrefix(text, axisInputPrefix):
		return Axis, strings.TrimPrefix(text, axisInputPrefix), true
	case strings.HasPrefix(text, hatInputPrefix):
		return Hat, strings.TrimPrefix(text, hatInputPrefix), true
	case strings.HasPrefix(text, buttonInputPrefix):
		return Button, strings.TrimPrefix(text, buttonInputPrefix), true
	}
	return "", "", false
}

// TemplateKeyFor returns the SVG template placeholder an input maps to:
// "Button_25", "AXIS_X", "POV_1_U". The "_OFF" suffix is stripped first, so
// BTN25_OFF and BTN25 share the placeholder Button_25.
func TemplateKeyFor(inputType InputType, inputID string) string {
	inputID = strings.TrimSuffix(inputID, "_OFF")

	switch inputType {
	case Button:
		return "Button_" + inputID
	case Axis:
		return "AXIS_" + strings.ToUpper(inputID)
	case Hat:
		return "POV_" + strings.ToUpper(inputID)
	}
	return ""
}
