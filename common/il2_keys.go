package common

import "strings"

// The names IL-2 writes for the keys that are not a plain letter or digit.
//
// Both the Gremlins and the TARGET enricher translate their own vocabulary into
// this one before matching a binding against the simulator's keyboard actions,
// and each used to carry its own copy of the table. The copies disagreed on
// Alt: Gremlins said "lalt" where IL-2 writes "lmenu", so every Alt combination
// coming from a Gremlins profile failed to find its action and fell back to the
// raw event label. TARGET contradicted itself too, emitting "lmenu" on the way
// out and expecting "lalt" on the way back.
//
// "lmenu" and "rmenu" are what the game actually writes: the IL-2 fixture in
// tests/integration holds 29 key_lmenu and 35 key_rmenu, and no key_lalt at all.
var il2KeyNames = map[string]string{
	"LShift":    "lshift",
	"RShift":    "rshift",
	"LCtrl":     "lcontrol",
	"RCtrl":     "rcontrol",
	"LAlt":      "lmenu",
	"RAlt":      "rmenu",
	"LWin":      "lwin",
	"RWin":      "rwin",
	"Space":     "space",
	"Enter":     "return",
	"Backspace": "back",
	"ESC":       "escape",
	"CapsLock":  "capital",
	"Tab":       "tab",
}

// IL2KeyName returns the IL-2 name of a key given its standard name, and
// whether the key is one of the special ones. Letters and digits are not in the
// table: they map to themselves in lower case.
func IL2KeyName(standard string) (string, bool) {
	name, found := il2KeyNames[standard]
	return name, found
}

// StandardKeyName is the reverse of IL2KeyName. An unknown name comes back
// upper-cased, which is what a plain letter or digit needs.
func StandardKeyName(il2Key string) string {
	for standard, name := range il2KeyNames {
		if name == il2Key {
			return standard
		}
	}
	return strings.ToUpper(il2Key)
}
