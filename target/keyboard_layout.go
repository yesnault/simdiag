package target

import "strings"

// KeyboardLayout represents a keyboard layout type
type KeyboardLayout string

const (
	KeyboardQWERTY KeyboardLayout = "qwerty"
	KeyboardAZERTY KeyboardLayout = "azerty"
)

// qwertyToAzerty maps QWERTY keys to their AZERTY equivalents
// These are the keys that differ between the two layouts
var qwertyToAzerty = map[string]string{
	// Letter keys that swap positions
	"Q": "A",
	"W": "Z",
	"A": "Q",
	"Z": "W",
	"M": ",",
	";": "M",

	// Lowercase versions
	"q": "a",
	"w": "z",
	"a": "q",
	"z": "w",
	"m": ",",

	// Special characters on number row (same physical position, different labels)
	// QWERTY "=" key position → AZERTY ")" key
	"=": ")",
	// QWERTY "-" key position → AZERTY ")" key (one position left)
	"-": ")",
}

// azertyToQwerty maps AZERTY keys to their QWERTY equivalents
var azertyToQwerty = map[string]string{
	// Letter keys that swap positions
	"A": "Q",
	"Z": "W",
	"Q": "A",
	"W": "Z",
	",": "M",
	"M": ";",

	// Lowercase versions
	"a": "q",
	"z": "w",
	"q": "a",
	"w": "z",

	// Number row - AZERTY characters to QWERTY numbers/symbols
	"&":  "1", // AZERTY & → QWERTY 1
	"é":  "2", // AZERTY é → QWERTY 2
	"É":  "2", // AZERTY É → QWERTY 2
	"\"": "3", // AZERTY " → QWERTY 3
	"'":  "4", // AZERTY ' → QWERTY 4
	"(":  "5", // AZERTY ( → QWERTY 5
	"§":  "6", // AZERTY § → QWERTY 6
	"è":  "7", // AZERTY è → QWERTY 7
	"È":  "7", // AZERTY È → QWERTY 7
	"!":  "8", // AZERTY ! → QWERTY 8
	"ç":  "9", // AZERTY ç → QWERTY 9
	"Ç":  "9", // AZERTY Ç → QWERTY 9
	"à":  "0", // AZERTY à → QWERTY 0
	"À":  "0", // AZERTY À → QWERTY 0
	"²":  "`", // AZERTY ² (top-left key) → QWERTY ` (backtick/grave)

	// Special characters on number row (same physical position, different labels)
	// AZERTY ")" key position → QWERTY "-" key (for IL-2 matching)
	")": "-",
}

// ConvertKeyForLayout converts a key name from one layout to another
// sourceLayout is the layout the key was defined in (e.g., TARGET uses QWERTY)
// targetLayout is the layout the simulator uses (e.g., IL-2 might use AZERTY)
func ConvertKeyForLayout(key string, sourceLayout, targetLayout KeyboardLayout) string {
	if sourceLayout == targetLayout {
		return key
	}

	// Normalize key for lookup (handle both "W" and "w")
	keyUpper := strings.ToUpper(key)

	if sourceLayout == KeyboardQWERTY && targetLayout == KeyboardAZERTY {
		if converted, ok := qwertyToAzerty[keyUpper]; ok {
			// Preserve original case
			if key == strings.ToLower(key) {
				return strings.ToLower(converted)
			}
			return converted
		}
	} else if sourceLayout == KeyboardAZERTY && targetLayout == KeyboardQWERTY {
		if converted, ok := azertyToQwerty[keyUpper]; ok {
			// Preserve original case
			if key == strings.ToLower(key) {
				return strings.ToLower(converted)
			}
			return converted
		}
	}

	return key
}

// ConvertKeysForLayout converts a slice of keys from one layout to another
func ConvertKeysForLayout(keys []string, sourceLayout, targetLayout KeyboardLayout) []string {
	if sourceLayout == targetLayout {
		return keys
	}

	converted := make([]string, len(keys))
	for i, key := range keys {
		converted[i] = ConvertKeyForLayout(key, sourceLayout, targetLayout)
	}
	return converted
}

// There is deliberately no way to read a layout from the configuration. The only
// keys that need converting come from a TARGET profile, and that profile already
// states the layout it was written in. See target.keyboardLayoutFromTarget.
