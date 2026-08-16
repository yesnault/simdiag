package common

import "testing"

// The names below are the ones the game itself writes. They are checked against
// tests/integration/testdata/full_export/fixtures/il2/global.actions, which is a
// real IL-2 configuration: it holds 29 key_lmenu and 35 key_rmenu, and no
// key_lalt or key_ralt anywhere.
func TestIL2KeyName_UsesTheNamesTheGameWrites(t *testing.T) {
	for standard, want := range map[string]string{
		"LAlt":   "lmenu",
		"RAlt":   "rmenu",
		"LCtrl":  "lcontrol",
		"RCtrl":  "rcontrol",
		"LShift": "lshift",
		"Enter":  "return",
		"ESC":    "escape",
	} {
		got, found := IL2KeyName(standard)
		if !found {
			t.Errorf("IL2KeyName(%q) reported the key as unknown", standard)
			continue
		}
		if got != want {
			t.Errorf("IL2KeyName(%q) = %q, want %q", standard, got, want)
		}
	}
}

// A letter or a digit is not in the table, and the caller lower-cases it itself.
func TestIL2KeyName_LeavesOrdinaryKeysAlone(t *testing.T) {
	for _, key := range []string{"A", "1", "F12"} {
		if got, found := IL2KeyName(key); found {
			t.Errorf("IL2KeyName(%q) = %q, want it left to the caller", key, got)
		}
	}
}

// The TARGET enricher converts a key to IL-2 and back while matching. The two
// directions used to live in separate tables that disagreed on Alt, so the round
// trip was not the identity.
func TestStandardKeyName_IsTheInverse(t *testing.T) {
	for standard := range il2KeyNames {
		il2Key, _ := IL2KeyName(standard)
		if got := StandardKeyName(il2Key); got != standard {
			t.Errorf("%q -> %q -> %q, want the original back", standard, il2Key, got)
		}
	}
}

func TestStandardKeyName_UpperCasesWhatItDoesNotKnow(t *testing.T) {
	if got := StandardKeyName("a"); got != "A" {
		t.Errorf("StandardKeyName(%q) = %q, want %q", "a", got, "A")
	}
}
