package gremlins

import (
	"testing"

	"simdiag/common"
)

// A Gremlins macro reaches its IL-2 action through three steps: the scan code
// becomes a key name, the key name becomes IL-2's spelling, and that spelling is
// looked up among the simulator's keyboard bindings. Nothing covered the chain,
// and it was broken in the middle: Gremlins translated Alt to "lalt" where IL-2
// writes "lmenu", so every Alt combination fell through to the raw event label.
//
// Verified on a real configuration (users/lelong, IL-2 with a Gremlins profile):
// the same binding read "LAlt + R" before the fix and "Reload turret guns" after.

// il2Keyboard builds the simulator side: keyboard bindings are the ones carrying
// DeviceGUID "keyboard", which is what findSimulatorActionForGremlins looks for.
func il2Keyboard(bindings map[string]string) []common.Binding {
	var out []common.Binding
	for inputID, description := range bindings {
		out = append(out, common.Binding{
			DeviceGUID:  "keyboard",
			InputID:     inputID,
			Description: description,
		})
	}
	return out
}

// buttonBinding returns the parsed binding for one button of the fixture.
func buttonBinding(t *testing.T, buttonID string) *Binding {
	t.Helper()

	parsed, err := ParseProfile("testdata/alt_binding.xml")
	if err != nil {
		t.Fatalf("ParseProfile: %v", err)
	}

	for _, b := range parsed {
		if b.InputID == buttonID {
			return b
		}
	}

	t.Fatalf("button %s is not in the fixture", buttonID)
	return nil
}

func TestFindSimulatorAction_MatchesAnAltCombination(t *testing.T) {
	binding := buttonBinding(t, "3")

	if binding.KeyboardKey != "LAlt + R" {
		t.Fatalf("the fixture parsed as %q, want %q", binding.KeyboardKey, "LAlt + R")
	}

	sim := il2Keyboard(map[string]string{"key_lmenu+key_r": "Reload turret guns"})

	if got := findSimulatorActionForGremlins(binding, sim); got != "Reload turret guns" {
		t.Errorf("findSimulatorActionForGremlins = %q, want the IL-2 action."+
			" An empty result means the Alt combination was spelled the way IL-2 does not.", got)
	}
}

// The Shift path always worked. Keeping it here is what makes the test above
// mean "Alt is fixed" rather than "the lookup works at all".
func TestFindSimulatorAction_MatchesAShiftCombination(t *testing.T) {
	binding := buttonBinding(t, "4")

	sim := il2Keyboard(map[string]string{"key_lshift+key_r": "Reload all guns"})

	if got := findSimulatorActionForGremlins(binding, sim); got != "Reload all guns" {
		t.Errorf("findSimulatorActionForGremlins = %q, want the IL-2 action", got)
	}
}

// IL-2 never writes lalt. A binding recorded under that name must not match, or
// the two spellings would both be accepted and the tables could drift again
// without any test noticing.
func TestFindSimulatorAction_DoesNotMatchTheOldAltSpelling(t *testing.T) {
	binding := buttonBinding(t, "3")

	sim := il2Keyboard(map[string]string{"key_lalt+key_r": "Reload turret guns"})

	if got := findSimulatorActionForGremlins(binding, sim); got != "" {
		t.Errorf("findSimulatorActionForGremlins = %q, want no match for key_lalt", got)
	}
}
