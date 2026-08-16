package common

import (
	"strings"
)

// NormalizeKeyNameForMatching normalizes key names for comparison between different systems
// Used by Gremlins, TARGET, and simulator parsers to match key bindings
func NormalizeKeyNameForMatching(key string) string {
	// Normalize common key name differences
	keyMappings := map[string]string{
		"Escape":     "ESC",
		"Return":     "Enter",
		"LControl":   "LCtrl",
		"RControl":   "RCtrl",
		"LShift":     "LShift",
		"RShift":     "RShift",
		"LAlt":       "LAlt",
		"RAlt":       "RAlt",
		"PageUp":     "PgUp",
		"PageDown":   "PgDn",
		"Backspace":  "Backspace",
		"Tab":        "Tab",
		"Space":      "Space",
		"CapsLock":   "CapsLock",
		"NumLock":    "NumLock",
		"ScrollLock": "ScrollLock",
	}

	if normalized, ok := keyMappings[key]; ok {
		return normalized
	}

	// Return uppercase version for consistency
	return strings.ToUpper(key)
}

// NormalizeKeyOrder normalizes the order of keys in a combination
// Modifiers (Ctrl, Shift, Alt, Win) come first in a standard order, followed by the main key
func NormalizeKeyOrder(keys []string) []string {
	if len(keys) <= 1 {
		return keys
	}

	// Separate modifiers from regular keys
	var modifiers []string
	var regularKeys []string

	// Define modifier priority order
	modifierPriority := map[string]int{
		"LCTRL":  1,
		"RCTRL":  2,
		"LSHIFT": 3,
		"RSHIFT": 4,
		"LALT":   5,
		"RALT":   6,
		"LWIN":   7,
		"RWIN":   8,
	}

	for _, key := range keys {
		keyUpper := strings.ToUpper(key)
		if _, isModifier := modifierPriority[keyUpper]; isModifier {
			modifiers = append(modifiers, key)
		} else {
			regularKeys = append(regularKeys, key)
		}
	}

	// Sort modifiers by priority
	for i := 0; i < len(modifiers); i++ {
		for j := i + 1; j < len(modifiers); j++ {
			if modifierPriority[strings.ToUpper(modifiers[i])] > modifierPriority[strings.ToUpper(modifiers[j])] {
				modifiers[i], modifiers[j] = modifiers[j], modifiers[i]
			}
		}
	}

	// Combine modifiers + regular keys
	result := make([]string, 0, len(keys))
	result = append(result, modifiers...)
	result = append(result, regularKeys...)

	return result
}
