package il2korea

import (
	"os"
	"path/filepath"
	"testing"

	"simdiag/common"
)

const testKnownDevices = `{
  "knownDevices": {
    "3833ad30-98c7-11f0-0000545345440180": {
      "deviceId": 8,
      "model": "Joystick Base",
      "ident": "4098_bea8",
      "isNew": false
    },
    "530648c0-98c7-11f0-0000545345440280": {
      "deviceId": 7,
      "model": "Throttle Base",
      "ident": "4098_bd64"
    },
    "a9f92800-3ee0-11f0-0000545345440380": {
      "deviceId": 6,
      "model": "Unused Box",
      "ident": "2342_8038"
    }
  }
}`

const testGeneralActions = `{
  "devices": [
    {"deviceId": 8, "model": "Joystick Base", "ident": "4098_bea8", "name": ""},
    {"deviceId": 7, "model": "Throttle Base", "ident": "4098_bd64", "name": ""}
  ],
  "modifiers": ["key_lshift"],
  "actions": {
    "all_guns_fire": ["key_space", "mouse_b0", "dev8_b3"],
    "gs_brightness": ["-dev7_axis_p"],
    "gs_yaw": ["dev8_axis_x"],
    "rpc_pitch_trim": ["dev8_pov0_180/dev8_pov0_0"],
    "unknown_device_action": ["dev99_b1"]
  }
}`

// writeInput writes the two Korea configuration files to a temp directory.
func writeInput(t *testing.T, knownDevices, generalActions string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, knownDevicesFileName), []byte(knownDevices), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", knownDevicesFileName, err)
	}
	if err := os.WriteFile(filepath.Join(dir, generalActionsFileName), []byte(generalActions), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", generalActionsFileName, err)
	}
	return dir
}

// findBinding returns the first binding matching an action name and input type.
func findBinding(bindings []common.Binding, action string, inputType common.InputType, inputID string) *common.Binding {
	for i := range bindings {
		b := bindings[i]
		if b.Action == action && b.InputType == inputType && b.InputID == inputID {
			return &bindings[i]
		}
	}
	return nil
}

func TestParseKnownDevices(t *testing.T) {
	dir := writeInput(t, testKnownDevices, testGeneralActions)

	devices, err := parseKnownDevices(filepath.Join(dir, knownDevicesFileName))
	if err != nil {
		t.Fatalf("parseKnownDevices failed: %v", err)
	}

	if len(devices) != 3 {
		t.Errorf("expected 3 devices, got %d", len(devices))
	}

	device, exists := devices[8]
	if !exists {
		t.Fatal("device 8 not found")
	}
	if device.GUID != "3833ad30-98c7-11f0-0000545345440180" {
		t.Errorf("device 8 GUID = %q, want the JSON object key", device.GUID)
	}
	if device.Name != "Joystick Base" {
		t.Errorf("device 8 name = %q, want %q", device.Name, "Joystick Base")
	}
}

func TestParseKnownDevicesErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		if _, err := parseKnownDevices(filepath.Join(t.TempDir(), "absent.json")); err == nil {
			t.Error("expected an error for a missing file")
		}
	})

	t.Run("empty device list", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, knownDevicesFileName)
		if err := os.WriteFile(path, []byte(`{"knownDevices":{}}`), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := parseKnownDevices(path); err == nil {
			t.Error("expected an error when no device is declared")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, knownDevicesFileName)
		if err := os.WriteFile(path, []byte(`{not json`), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := parseKnownDevices(path); err == nil {
			t.Error("expected an error for malformed JSON")
		}
	})
}

// TestParseRoster checks that the profile only carries devices declared in
// general.actions, not every device known to the game.
func TestParseRoster(t *testing.T) {
	dir := writeInput(t, testKnownDevices, testGeneralActions)

	collection, err := NewParser(nil).Parse(dir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	profile := collection.Profiles[0]
	if len(profile.Devices) != 2 {
		t.Errorf("expected 2 devices in profile, got %d", len(profile.Devices))
	}
	if _, exists := profile.Devices["a9f92800-3ee0-11f0-0000545345440380"]; exists {
		t.Error("device absent from general.actions should not be in the profile")
	}
	if profile.SimType != common.IL2Korea {
		t.Errorf("SimType = %v, want %v", profile.SimType, common.IL2Korea)
	}
}

// TestParseBindingConversions covers the reference formats: 0-based buttons, axis
// letters, POV angles, inversion prefix, "/" splitting, keyboard and mouse handling.
func TestParseBindingConversions(t *testing.T) {
	dir := writeInput(t, testKnownDevices, testGeneralActions)

	collection, err := NewParser(nil).Parse(dir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	bindings := collection.Profiles[0].Bindings

	tests := []struct {
		name       string
		action     string
		inputType  common.InputType
		inputID    string
		wantDevice string
	}{
		{"button is converted from 0-based to 1-based", "all_guns_fire", common.Button, "4", "Joystick Base"},
		{"axis letter p maps to SLIDER_1 despite the inversion prefix", "gs_brightness", common.Axis, "SLIDER_1", "Throttle Base"},
		{"axis letter x maps to X", "gs_yaw", common.Axis, "X", "Joystick Base"},
		{"POV angle 180 maps to direction D on hat 1", "rpc_pitch_trim", common.Hat, "1_D", "Joystick Base"},
		{"second half of a / pair is kept", "rpc_pitch_trim", common.Hat, "1_U", "Joystick Base"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binding := findBinding(bindings, tt.action, tt.inputType, tt.inputID)
			if binding == nil {
				t.Fatalf("no binding for action %q with input %s %q", tt.action, tt.inputType, tt.inputID)
			}
			if binding.DeviceName != tt.wantDevice {
				t.Errorf("device = %q, want %q", binding.DeviceName, tt.wantDevice)
			}
		})
	}

	t.Run("keyboard bindings are kept", func(t *testing.T) {
		if findBinding(bindings, "all_guns_fire", common.Button, "key_space") == nil {
			t.Error("keyboard binding key_space missing")
		}
	})

	t.Run("mouse bindings are skipped", func(t *testing.T) {
		for _, b := range bindings {
			if b.InputID == "mouse_b0" {
				t.Error("mouse binding should not produce a binding")
			}
		}
	})

	t.Run("references to unknown devices are skipped", func(t *testing.T) {
		for _, b := range bindings {
			if b.Action == "unknown_device_action" {
				t.Errorf("binding on unknown device dev99 should be skipped, got %+v", b)
			}
		}
	})
}

// TestDescriptionFallback checks that without a Great Battles installation the raw
// action name is used as the display text.
func TestDescriptionFallback(t *testing.T) {
	dir := writeInput(t, testKnownDevices, testGeneralActions)

	collection, err := NewParser(nil).Parse(dir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	binding := findBinding(collection.Profiles[0].Bindings, "all_guns_fire", common.Button, "4")
	if binding == nil {
		t.Fatal("binding not found")
	}
	if binding.Description != "all_guns_fire" {
		t.Errorf("Description = %q, want the raw action name %q", binding.Description, "all_guns_fire")
	}
}

// TestDescriptionFromGreatBattles checks that labels are borrowed from a configured
// Great Battles installation, and that unknown actions still fall back to their name.
func TestDescriptionFromGreatBattles(t *testing.T) {
	koreaDir := writeInput(t, testKnownDevices, testGeneralActions)

	gbDir := t.TempDir()
	globalActions := "all_guns_fire, joy1_b0, 0| // Tirer toutes les armes\n" +
		"gs_yaw, joy1_axis_x, 0| // Viseur : lacet\n"
	if err := os.WriteFile(filepath.Join(gbDir, globalActionsFileName), []byte(globalActions), 0644); err != nil {
		t.Fatal(err)
	}

	config := &common.Config{
		Simulators: map[string]*common.SimulatorConfig{
			common.IL2Sturmovik.GetConfigKey(): {IL2InputPath: gbDir},
		},
	}

	collection, err := NewParser(config).Parse(koreaDir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	bindings := collection.Profiles[0].Bindings

	binding := findBinding(bindings, "all_guns_fire", common.Button, "4")
	if binding == nil {
		t.Fatal("binding not found")
	}
	if binding.Description != "Tirer toutes les armes" {
		t.Errorf("Description = %q, want the Great Battles label", binding.Description)
	}

	// gs_brightness has no Great Battles label in this fixture
	fallback := findBinding(bindings, "gs_brightness", common.Axis, "SLIDER_1")
	if fallback == nil {
		t.Fatal("gs_brightness binding not found")
	}
	if fallback.Description != "gs_brightness" {
		t.Errorf("Description = %q, want the raw action name for an action absent from Great Battles", fallback.Description)
	}
}

// TestParseIsDeterministic guards the sorted iteration over the actions map.
func TestParseIsDeterministic(t *testing.T) {
	dir := writeInput(t, testKnownDevices, testGeneralActions)
	parser := NewParser(nil)

	first, err := parser.Parse(dir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		next, err := parser.Parse(dir)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if len(next.Profiles[0].Bindings) != len(first.Profiles[0].Bindings) {
			t.Fatalf("binding count changed between runs")
		}
		for j, b := range next.Profiles[0].Bindings {
			want := first.Profiles[0].Bindings[j]
			if b.Action != want.Action || b.InputType != want.InputType || b.InputID != want.InputID || b.DeviceGUID != want.DeviceGUID {
				t.Fatalf("binding order changed between runs at index %d: got %s/%s/%s, want %s/%s/%s",
					j, b.Action, b.InputType, b.InputID, want.Action, want.InputType, want.InputID)
			}
		}
	}
}
