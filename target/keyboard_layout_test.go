package target

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"simdiag/common"
)

// TestKeyboardLayoutFromTarget pins the one value whose meaning is known, and the
// conservative default for the rest: anything unrecognised must behave exactly as
// it did before the layout was read from the file at all.
func TestKeyboardLayoutFromTarget(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  KeyboardLayout
	}{
		{"1 is AZERTY", 1, KeyboardAZERTY},
		{"0 falls back", 0, KeyboardQWERTY},
		{"2 falls back", 2, KeyboardQWERTY},
		{"an unknown value falls back", 42, KeyboardQWERTY},
		{"a negative value falls back", -1, KeyboardQWERTY},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := keyboardLayoutFromTarget(tt.value); got != tt.want {
				t.Errorf("keyboardLayoutFromTarget(%d) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

// writeFCF writes a minimal TARGET profile declaring the given layout and binding
// one key on device 1001.
func writeFCF(t *testing.T, layout int, key string) string {
	t.Helper()

	content := `<?xml version="1.0" encoding="utf-8"?>
<FastEventsMapping>
  <Version><ProgramVersionNumber>3.0.21.910</ProgramVersionNumber></Version>
  <ProjectData>
    <KeyboardLayout>` + strconv.Itoa(layout) + `</KeyboardLayout>
    <SelectedDevices>1001 </SelectedDevices>
  </ProjectData>
  <EventsList>
    <Event0>
      <HidEvent>
        <DeviceNumber>1001</DeviceNumber>
        <Name>TG1</Name>
        <HidType>3</HidType>
        <EventType>1</EventType>
        <ActionType>1</ActionType>
        <ControlIndex>1</ControlIndex>
        <Events>
          <HidCommand0>
            <EventName>test</EventName>
            <Layers>1 16 32 64</Layers>
            <EventsNumber>1</EventsNumber>
            <HidEvent0>
              <DeviceNumber>-1</DeviceNumber>
              <Name>` + key + `</Name>
              <HidType>1</HidType>
              <EventType>1</EventType>
            </HidEvent0>
          </HidCommand0>
        </Events>
      </HidEvent>
    </Event0>
  </EventsList>
</FastEventsMapping>`

	path := filepath.Join(t.TempDir(), "profile.fcf")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return path
}

// The layout the profile declares must reach every binding parsed from it:
// that is the whole point of reading it instead of asking the user.
func TestParseProfile_StampsTheDeclaredLayout(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value int
		want  KeyboardLayout
	}{
		{"an AZERTY profile", 1, KeyboardAZERTY},
		{"a profile declaring 0", 0, KeyboardQWERTY},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bindings, err := ParseProfile(writeFCF(t, tt.value, "é"))
			if err != nil {
				t.Fatalf("ParseProfile: %v", err)
			}
			if len(bindings) == 0 {
				t.Fatal("expected at least one binding from the fixture")
			}

			for _, b := range bindings {
				if b.KeyboardLayout != tt.want {
					t.Errorf("binding %q layout = %q, want %q", b.InputName, b.KeyboardLayout, tt.want)
				}
			}
		})
	}
}

// The end the layout serves: an AZERTY "é" must match the simulator's QWERTY "2",
// and must not when the profile says it was written on a QWERTY keyboard.
func TestFindSimulatorActionForTarget_UsesTheProfileLayout(t *testing.T) {
	simBindings := []common.Binding{
		{DeviceGUID: "keyboard", InputID: "2", Description: "Fusée verte"},
	}

	azerty := &Binding{OutputKeys: []string{"é"}, KeyboardLayout: KeyboardAZERTY}
	if got := findSimulatorActionForTarget(azerty, simBindings); len(got) == 0 || got[0] != "Fusée verte" {
		t.Errorf("an AZERTY \"é\" should resolve to the action bound to \"2\", got %v", got)
	}

	qwerty := &Binding{OutputKeys: []string{"é"}, KeyboardLayout: KeyboardQWERTY}
	if got := findSimulatorActionForTarget(qwerty, simBindings); len(got) != 0 {
		t.Errorf("a QWERTY profile must not convert \"é\", got %v", got)
	}
}
