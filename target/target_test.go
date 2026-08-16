package target

import (
	"simdiag/common"
	"testing"
)

// TestGetProfilePath: a TARGET script is written for a physical HOTAS, so it is
// configured once per simulator and applies to every DCS module.
func TestGetProfilePath(t *testing.T) {
	const dcsProfile = `C:\target\all.tmc`

	config := &common.Config{Simulators: map[string]*common.SimulatorConfig{
		"dcs_world":     {DCSPath: `C:\DCS`, TargetProfileFilepath: dcsProfile},
		"il2_sturmovik": {IL2InputPath: `C:\IL-2`},
	}}

	if got := GetProfilePath(config, common.DCSWorld); got != dcsProfile {
		t.Errorf("GetProfilePath(DCS) = %q, want %q", got, dcsProfile)
	}
	if got := GetProfilePath(config, common.IL2Sturmovik); got != "" {
		t.Errorf("GetProfilePath(IL-2) = %q, want empty", got)
	}
	if got := GetProfilePath(config, common.IL2Korea); got != "" {
		t.Errorf("GetProfilePath(unconfigured) = %q, want empty", got)
	}
	if got := GetProfilePath(nil, common.DCSWorld); got != "" {
		t.Errorf("GetProfilePath(nil config) = %q, want empty", got)
	}
}

// TestParseButtonID tests the button ID parsing logic
func TestParseButtonID(t *testing.T) {
	tests := []struct {
		name          string
		buttonID      string
		wantInputType common.InputType
		wantInputID   string
	}{
		{
			name:          "button 1",
			buttonID:      "BTN1",
			wantInputType: common.Button,
			wantInputID:   "1",
		},
		{
			name:          "button 25",
			buttonID:      "BTN25",
			wantInputType: common.Button,
			wantInputID:   "25",
		},
		{
			name:          "POV up",
			buttonID:      "POV_1_U",
			wantInputType: common.Hat,
			wantInputID:   "1_U",
		},
		{
			name:          "POV down",
			buttonID:      "POV_1_D",
			wantInputType: common.Hat,
			wantInputID:   "1_D",
		},
		{
			name:          "POV left",
			buttonID:      "POV_1_L",
			wantInputType: common.Hat,
			wantInputID:   "1_L",
		},
		{
			name:          "POV right",
			buttonID:      "POV_1_R",
			wantInputType: common.Hat,
			wantInputID:   "1_R",
		},
		{
			name:          "axis X",
			buttonID:      "AXIS_X",
			wantInputType: common.Axis,
			wantInputID:   "X",
		},
		{
			name:          "axis Y",
			buttonID:      "AXIS_Y",
			wantInputType: common.Axis,
			wantInputID:   "Y",
		},
		{
			name:          "axis SLIDER",
			buttonID:      "AXIS_SLIDER",
			wantInputType: common.Axis,
			wantInputID:   "SLIDER",
		},
		{
			name:          "unknown format defaults to button",
			buttonID:      "UNKNOWN",
			wantInputType: common.Button,
			wantInputID:   "",
		},
		{
			name:          "empty string",
			buttonID:      "",
			wantInputType: common.Button,
			wantInputID:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInputType, gotInputID := parseButtonID(tt.buttonID)
			if gotInputType != tt.wantInputType {
				t.Errorf("parseButtonID(%q) inputType = %v, want %v", tt.buttonID, gotInputType, tt.wantInputType)
			}
			if gotInputID != tt.wantInputID {
				t.Errorf("parseButtonID(%q) inputID = %v, want %v", tt.buttonID, gotInputID, tt.wantInputID)
			}
		})
	}
}

// TestDetermineAction tests the action determination based on trigger type
func TestDetermineAction(t *testing.T) {
	tests := []struct {
		name    string
		trigger string
		want    string
	}{
		{
			name:    "trigger I",
			trigger: "I",
			want:    "TARGET_I",
		},
		{
			name:    "trigger O",
			trigger: "O",
			want:    "TARGET_O",
		},
		{
			name:    "trigger P (default)",
			trigger: "P",
			want:    "TARGET",
		},
		{
			name:    "empty trigger (default)",
			trigger: "",
			want:    "TARGET",
		},
		{
			name:    "unknown trigger (default)",
			trigger: "X",
			want:    "TARGET",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := &Binding{Trigger: tt.trigger}
			got := determineAction(tb)
			if got != tt.want {
				t.Errorf("determineAction(trigger=%q) = %v, want %v", tt.trigger, got, tt.want)
			}
		})
	}
}

// TestTargetKeyToIL2Key tests the TARGET to IL-2 key name conversion
func TestTargetKeyToIL2Key(t *testing.T) {
	tests := []struct {
		name      string
		targetKey string
		want      string
	}{
		// Modifiers
		{
			name:      "left shift",
			targetKey: "LShift",
			want:      "lshift",
		},
		{
			name:      "right shift",
			targetKey: "RShift",
			want:      "rshift",
		},
		{
			name:      "left control",
			targetKey: "LCtrl",
			want:      "lcontrol",
		},
		{
			name:      "right control",
			targetKey: "RCtrl",
			want:      "rcontrol",
		},
		{
			name:      "left alt",
			targetKey: "LAlt",
			want:      "lmenu",
		},
		{
			name:      "right alt",
			targetKey: "RAlt",
			want:      "rmenu",
		},
		// Special keys
		{
			name:      "space",
			targetKey: "Space",
			want:      "space",
		},
		{
			name:      "enter",
			targetKey: "Enter",
			want:      "return",
		},
		{
			name:      "backspace",
			targetKey: "Backspace",
			want:      "back",
		},
		{
			name:      "escape",
			targetKey: "ESC",
			want:      "escape",
		},
		{
			name:      "tab",
			targetKey: "Tab",
			want:      "tab",
		},
		// Numpad keys
		{
			name:      "numpad 0",
			targetKey: "Num0",
			want:      "numpad0",
		},
		{
			name:      "numpad enter",
			targetKey: "NumEnter",
			want:      "numpadenter",
		},
		{
			name:      "numpad plus",
			targetKey: "Num+",
			want:      "add",
		},
		{
			name:      "numpad minus",
			targetKey: "Num-",
			want:      "subtract",
		},
		// Special characters
		{
			name:      "equals",
			targetKey: "=",
			want:      "equals",
		},
		{
			name:      "minus",
			targetKey: "-",
			want:      "minus",
		},
		{
			name:      "AZERTY closing parenthesis (equals key)",
			targetKey: ")",
			want:      "equals",
		},
		{
			name:      "AZERTY opening parenthesis (minus key)",
			targetKey: "(",
			want:      "minus",
		},
		{
			name:      "left bracket",
			targetKey: "[",
			want:      "lbracket",
		},
		{
			name:      "semicolon",
			targetKey: ";",
			want:      "semicolon",
		},
		// Regular keys (lowercase)
		{
			name:      "letter A",
			targetKey: "A",
			want:      "a",
		},
		{
			name:      "letter Z",
			targetKey: "Z",
			want:      "z",
		},
		{
			name:      "number 1",
			targetKey: "1",
			want:      "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := targetKeyToIL2Key(tt.targetKey)
			if got != tt.want {
				t.Errorf("targetKeyToIL2Key(%q) = %v, want %v", tt.targetKey, got, tt.want)
			}
		})
	}
}

// TestIL2KeyToStandard tests the IL-2 to standard key name conversion
func TestIL2KeyToStandard(t *testing.T) {
	tests := []struct {
		name   string
		il2Key string
		want   string
	}{
		// Modifiers
		{
			name:   "left shift",
			il2Key: "lshift",
			want:   "LShift",
		},
		{
			name:   "right shift",
			il2Key: "rshift",
			want:   "RShift",
		},
		{
			name:   "left control",
			il2Key: "lcontrol",
			want:   "LCtrl",
		},
		{
			name:   "right control",
			il2Key: "rcontrol",
			want:   "RCtrl",
		},
		// IL-2 writes lmenu and rmenu for the Alt keys, never lalt or ralt: the
		// integration fixture holds 29 key_lmenu and 35 key_rmenu and no
		// key_lalt at all. This test used to assert the names the Gremlins
		// enricher had invented, which is how the two tables drifted apart.
		{
			name:   "left alt",
			il2Key: "lmenu",
			want:   "LAlt",
		},
		{
			name:   "right alt",
			il2Key: "rmenu",
			want:   "RAlt",
		},
		// Special keys
		{
			name:   "space",
			il2Key: "space",
			want:   "Space",
		},
		{
			name:   "return (enter)",
			il2Key: "return",
			want:   "Enter",
		},
		{
			name:   "back (backspace)",
			il2Key: "back",
			want:   "Backspace",
		},
		{
			name:   "escape",
			il2Key: "escape",
			want:   "ESC",
		},
		{
			name:   "capital (caps lock)",
			il2Key: "capital",
			want:   "CapsLock",
		},
		// Unmapped keys (uppercase)
		{
			name:   "letter a",
			il2Key: "a",
			want:   "A",
		},
		{
			name:   "number 1",
			il2Key: "1",
			want:   "1",
		},
		{
			name:   "equals",
			il2Key: "equals",
			want:   "EQUALS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := il2KeyToStandard(tt.il2Key)
			if got != tt.want {
				t.Errorf("il2KeyToStandard(%q) = %v, want %v", tt.il2Key, got, tt.want)
			}
		})
	}
}

// TestConvertTargetToIL2Format tests the complete TARGET to IL-2 format conversion
func TestConvertTargetToIL2Format(t *testing.T) {
	tests := []struct {
		name string
		keys []string
		want string
	}{
		{
			name: "single key",
			keys: []string{"A"},
			want: "key_a",
		},
		{
			name: "modifier + key",
			keys: []string{"LShift", "A"},
			want: "key_lshift+key_a",
		},
		{
			name: "multiple modifiers",
			keys: []string{"LCtrl", "LAlt", "Space"},
			want: "key_lcontrol+key_lmenu+key_space",
		},
		{
			name: "AZERTY special char",
			keys: []string{"LShift", ")"},
			want: "key_lshift+key_equals",
		},
		{
			name: "numpad key",
			keys: []string{"Num+"},
			want: "key_add",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertTargetToIL2Format(tt.keys)
			if got != tt.want {
				t.Errorf("convertTargetToIL2Format(%v) = %v, want %v", tt.keys, got, tt.want)
			}
		})
	}
}

// TestNormalizeTargetKeys tests the TARGET key normalization
func TestNormalizeTargetKeys(t *testing.T) {
	tests := []struct {
		name string
		keys []string
		want string
	}{
		{
			name: "single lowercase key",
			keys: []string{"a"},
			want: "A",
		},
		{
			name: "multiple keys",
			keys: []string{"lshift", "a"},
			want: "LSHIFT + A",
		},
		{
			name: "mixed case",
			keys: []string{"LShift", "a"},
			want: "LShift + A",
		},
		{
			name: "special characters",
			keys: []string{"LCtrl", "+"},
			want: "LCTRL + +",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeTargetKeys(tt.keys)
			if got != tt.want {
				t.Errorf("normalizeTargetKeys(%v) = %v, want %v", tt.keys, got, tt.want)
			}
		})
	}
}

// TestIsMidPosition tests the mid position detection
func TestIsMidPosition(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "boat switch middle",
			input: "BSM",
			want:  true,
		},
		{
			name:  "flaps middle",
			input: "FLAPM",
			want:  true,
		},
		{
			name:  "pinky switch middle",
			input: "PSM",
			want:  true,
		},
		{
			name:  "boat switch middle lowercase",
			input: "bsm",
			want:  true,
		},
		{
			name:  "not a mid position",
			input: "BSF",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "random string",
			input: "RANDOM",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMidPosition(tt.input)
			if got != tt.want {
				t.Errorf("IsMidPosition(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestDeviceNumberToName tests the device number to name conversion
func TestDeviceNumberToName(t *testing.T) {
	tests := []struct {
		name         string
		deviceNumber int
		want         string
	}{
		{
			name:         "joystick",
			deviceNumber: 1001,
			want:         "Joystick",
		},
		{
			name:         "throttle",
			deviceNumber: 1002,
			want:         "Throttle",
		},
		{
			name:         "MFD",
			deviceNumber: 1003,
			want:         "MFD",
		},
		{
			name:         "rudder pedals",
			deviceNumber: 1004,
			want:         "Rudder Pedals",
		},
		{
			name:         "device 5",
			deviceNumber: 1005,
			want:         "Device 5",
		},
		{
			name:         "device 6",
			deviceNumber: 1006,
			want:         "Device 6",
		},
		{
			name:         "unknown device number",
			deviceNumber: 9999,
			want:         "Device 9999",
		},
		{
			name:         "zero",
			deviceNumber: 0,
			want:         "Device 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeviceNumberToName(tt.deviceNumber)
			if got != tt.want {
				t.Errorf("DeviceNumberToName(%d) = %v, want %v", tt.deviceNumber, got, tt.want)
			}
		})
	}
}
