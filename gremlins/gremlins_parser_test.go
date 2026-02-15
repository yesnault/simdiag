package gremlins

import (
	"os"
	"path/filepath"
	"simdiag/common"
	"testing"
)

// TestScanCodeToKeyName tests the Windows scan code to key name conversion
func TestScanCodeToKeyName(t *testing.T) {
	tests := []struct {
		name     string
		scanCode int
		extended bool
		want     string
	}{
		// Basic keys
		{
			name:     "escape",
			scanCode: 1,
			extended: false,
			want:     "ESC",
		},
		{
			name:     "number 1",
			scanCode: 2,
			extended: false,
			want:     "1",
		},
		{
			name:     "letter A",
			scanCode: 30,
			extended: false,
			want:     "A",
		},
		{
			name:     "letter Z",
			scanCode: 44,
			extended: false,
			want:     "Z",
		},
		{
			name:     "space",
			scanCode: 57,
			extended: false,
			want:     "Space",
		},
		// Modifier keys
		{
			name:     "left shift",
			scanCode: 42,
			extended: false,
			want:     "LShift",
		},
		{
			name:     "right shift",
			scanCode: 54,
			extended: false,
			want:     "RShift",
		},
		{
			name:     "left control",
			scanCode: 29,
			extended: false,
			want:     "LCtrl",
		},
		{
			name:     "right control (extended)",
			scanCode: 29,
			extended: true,
			want:     "RCtrl",
		},
		{
			name:     "left alt",
			scanCode: 56,
			extended: false,
			want:     "LAlt",
		},
		{
			name:     "right alt (extended)",
			scanCode: 56,
			extended: true,
			want:     "RAlt",
		},
		// Function keys
		{
			name:     "F1",
			scanCode: 59,
			extended: false,
			want:     "F1",
		},
		{
			name:     "F10",
			scanCode: 68,
			extended: false,
			want:     "F10",
		},
		{
			name:     "F11",
			scanCode: 87,
			extended: false,
			want:     "F11",
		},
		{
			name:     "F12",
			scanCode: 88,
			extended: false,
			want:     "F12",
		},
		// Numpad keys
		{
			name:     "numpad 0",
			scanCode: 82,
			extended: false,
			want:     "Num0",
		},
		{
			name:     "numpad 9",
			scanCode: 73,
			extended: false,
			want:     "Num9",
		},
		{
			name:     "numpad plus",
			scanCode: 78,
			extended: false,
			want:     "Num+",
		},
		{
			name:     "numpad minus",
			scanCode: 74,
			extended: false,
			want:     "Num-",
		},
		{
			name:     "numpad enter (extended)",
			scanCode: 28,
			extended: true,
			want:     "NumEnter",
		},
		{
			name:     "numpad divide (extended)",
			scanCode: 53,
			extended: true,
			want:     "Num/",
		},
		// Special keys
		{
			name:     "enter",
			scanCode: 28,
			extended: false,
			want:     "Enter",
		},
		{
			name:     "backspace",
			scanCode: 14,
			extended: false,
			want:     "Backspace",
		},
		{
			name:     "tab",
			scanCode: 15,
			extended: false,
			want:     "Tab",
		},
		// Extended navigation keys
		{
			name:     "home (extended)",
			scanCode: 71,
			extended: true,
			want:     "Home",
		},
		{
			name:     "end (extended)",
			scanCode: 79,
			extended: true,
			want:     "End",
		},
		{
			name:     "page up (extended)",
			scanCode: 73,
			extended: true,
			want:     "PgUp",
		},
		{
			name:     "page down (extended)",
			scanCode: 81,
			extended: true,
			want:     "PgDn",
		},
		{
			name:     "arrow up (extended)",
			scanCode: 72,
			extended: true,
			want:     "Up",
		},
		{
			name:     "arrow down (extended)",
			scanCode: 80,
			extended: true,
			want:     "Down",
		},
		{
			name:     "arrow left (extended)",
			scanCode: 75,
			extended: true,
			want:     "Left",
		},
		{
			name:     "arrow right (extended)",
			scanCode: 77,
			extended: true,
			want:     "Right",
		},
		{
			name:     "insert (extended)",
			scanCode: 82,
			extended: true,
			want:     "Insert",
		},
		{
			name:     "delete (extended)",
			scanCode: 83,
			extended: true,
			want:     "Delete",
		},
		{
			name:     "left windows key (extended)",
			scanCode: 91,
			extended: true,
			want:     "LWin",
		},
		{
			name:     "right windows key (extended)",
			scanCode: 92,
			extended: true,
			want:     "RWin",
		},
		// Unknown scan code
		{
			name:     "unknown scan code",
			scanCode: 999,
			extended: false,
			want:     "Key999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanCodeToKeyName(tt.scanCode, tt.extended)
			if got != tt.want {
				t.Errorf("scanCodeToKeyName(%d, %v) = %v, want %v", tt.scanCode, tt.extended, got, tt.want)
			}
		})
	}
}

// TestConvertAxisIDToName tests the Gremlins axis ID to standard axis name conversion
func TestConvertAxisIDToName(t *testing.T) {
	tests := []struct {
		name   string
		axisID string
		want   string
	}{
		{
			name:   "axis 1 (X)",
			axisID: "1",
			want:   "X",
		},
		{
			name:   "axis 2 (Y)",
			axisID: "2",
			want:   "Y",
		},
		{
			name:   "axis 3 (Z)",
			axisID: "3",
			want:   "Z",
		},
		{
			name:   "axis 4 (RY)",
			axisID: "4",
			want:   "RY",
		},
		{
			name:   "axis 5 (RX)",
			axisID: "5",
			want:   "RX",
		},
		{
			name:   "axis 6 (RZ)",
			axisID: "6",
			want:   "RZ",
		},
		{
			name:   "axis 7 (SLIDER_1)",
			axisID: "7",
			want:   "SLIDER_1",
		},
		{
			name:   "axis 8 (SLIDER_2)",
			axisID: "8",
			want:   "SLIDER_2",
		},
		{
			name:   "unknown axis (returns as-is)",
			axisID: "9",
			want:   "9",
		},
		{
			name:   "custom axis name",
			axisID: "CUSTOM",
			want:   "CUSTOM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertAxisIDToName(tt.axisID)
			if got != tt.want {
				t.Errorf("convertAxisIDToName(%q) = %v, want %v", tt.axisID, got, tt.want)
			}
		})
	}
}

// TestParseProfile tests the Gremlins XML profile parsing
func TestParseProfile(t *testing.T) {
	tests := []struct {
		name             string
		xmlContent       string
		wantBindingCount int
		wantFirstType    common.InputType
		wantFirstID      string
		wantFirstMode    string
		wantError        bool
	}{
		{
			name: "simple button with keyboard mapping",
			xmlContent: `<?xml version="1.0"?>
<profile version="1">
    <devices>
        <device device-guid="{EE6F1C30-3F2E-11F0-8001-444553540000}" name="Test Joystick" type="joystick">
            <mode name="Base">
                <button id="1" description="Trigger">
                    <container>
                        <action-set>
                            <map-to-keyboard>
                                <key scan-code="57" extended="false"/>
                            </map-to-keyboard>
                        </action-set>
                    </container>
                </button>
            </mode>
        </device>
    </devices>
</profile>`,
			wantBindingCount: 1,
			wantFirstType:    common.Button,
			wantFirstID:      "1",
			wantFirstMode:    "Base",
			wantError:        false,
		},
		{
			name: "button with vJoy remap",
			xmlContent: `<?xml version="1.0"?>
<profile version="1">
    <devices>
        <device device-guid="{EE6F1C30-3F2E-11F0-8001-444553540000}" name="Test Joystick" type="joystick">
            <mode name="Base">
                <button id="25" description="Test Button">
                    <container>
                        <action-set>
                            <remap button="98" axis="0" hat="0" vjoy="3"/>
                        </action-set>
                    </container>
                </button>
            </mode>
        </device>
    </devices>
</profile>`,
			wantBindingCount: 1,
			wantFirstType:    common.Button,
			wantFirstID:      "25",
			wantFirstMode:    "Base",
			wantError:        false,
		},
		{
			name: "button with temporary mode switch",
			xmlContent: `<?xml version="1.0"?>
<profile version="1">
    <devices>
        <device device-guid="{EE6F1C30-3F2E-11F0-8001-444553540000}" name="Test Joystick" type="joystick">
            <mode name="Base">
                <button id="10" description="Shift Button">
                    <container>
                        <action-set>
                            <temporary-mode-switch name="Shift"/>
                        </action-set>
                    </container>
                </button>
            </mode>
        </device>
    </devices>
</profile>`,
			wantBindingCount: 1,
			wantFirstType:    common.Button,
			wantFirstID:      "10",
			wantFirstMode:    "Base",
			wantError:        false,
		},
		{
			name: "axis with vJoy remap",
			xmlContent: `<?xml version="1.0"?>
<profile version="1">
    <devices>
        <device device-guid="{EE6F1C30-3F2E-11F0-8001-444553540000}" name="Test Joystick" type="joystick">
            <mode name="Base">
                <axis id="1" description="X Axis">
                    <container>
                        <action-set>
                            <remap button="0" axis="1" hat="0" vjoy="3"/>
                        </action-set>
                    </container>
                </axis>
            </mode>
        </device>
    </devices>
</profile>`,
			wantBindingCount: 1,
			wantFirstType:    common.Axis,
			wantFirstID:      "X",
			wantFirstMode:    "Base",
			wantError:        false,
		},
		{
			name: "hat with vJoy remap (creates 4 bindings)",
			xmlContent: `<?xml version="1.0"?>
<profile version="1">
    <devices>
        <device device-guid="{EE6F1C30-3F2E-11F0-8001-444553540000}" name="Test Joystick" type="joystick">
            <mode name="Base">
                <hat id="1" description="Hat Switch">
                    <container>
                        <action-set>
                            <remap button="0" axis="0" hat="1" vjoy="3"/>
                        </action-set>
                    </container>
                </hat>
            </mode>
        </device>
    </devices>
</profile>`,
			wantBindingCount: 4, // U, D, L, R
			wantFirstType:    common.Hat,
			wantFirstID:      "1_U",
			wantFirstMode:    "Base",
			wantError:        false,
		},
		{
			name: "multiple devices and modes",
			xmlContent: `<?xml version="1.0"?>
<profile version="1">
    <devices>
        <device device-guid="{EE6F1C30-3F2E-11F0-8001-444553540000}" name="Joystick" type="joystick">
            <mode name="Base">
                <button id="1" description="Button 1">
                    <container>
                        <action-set>
                            <map-to-keyboard>
                                <key scan-code="57" extended="false"/>
                            </map-to-keyboard>
                        </action-set>
                    </container>
                </button>
            </mode>
            <mode name="Shift">
                <button id="1" description="Button 1 Shifted">
                    <container>
                        <action-set>
                            <map-to-keyboard>
                                <key scan-code="59" extended="false"/>
                            </map-to-keyboard>
                        </action-set>
                    </container>
                </button>
            </mode>
        </device>
        <device device-guid="{A7C91C00-3F30-11F0-8001-444553540000}" name="Throttle" type="joystick">
            <mode name="Base">
                <button id="16" description="Throttle Button">
                    <container>
                        <action-set>
                            <map-to-keyboard>
                                <key scan-code="1" extended="false"/>
                            </map-to-keyboard>
                        </action-set>
                    </container>
                </button>
            </mode>
        </device>
    </devices>
</profile>`,
			wantBindingCount: 3,
			wantError:        false,
		},
		{
			name:             "empty profile",
			xmlContent:       `<?xml version="1.0"?><profile version="1"><devices></devices></profile>`,
			wantBindingCount: 0,
			wantError:        false,
		},
		{
			name:       "invalid XML",
			xmlContent: `<?xml version="1.0"?><profile><unclosed>`,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "profile.xml")
			err := os.WriteFile(tmpFile, []byte(tt.xmlContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Test the function
			bindings, err := ParseProfile(tmpFile)

			if (err != nil) != tt.wantError {
				t.Errorf("ParseProfile() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil {
				return // Error was expected
			}

			if len(bindings) != tt.wantBindingCount {
				t.Errorf("ParseProfile() returned %d bindings, want %d", len(bindings), tt.wantBindingCount)
			}

			if tt.wantBindingCount > 0 && len(bindings) > 0 {
				first := bindings[0]
				if tt.wantFirstType != "" && first.InputType != tt.wantFirstType {
					t.Errorf("First binding InputType = %v, want %v", first.InputType, tt.wantFirstType)
				}
				if tt.wantFirstID != "" && first.InputID != tt.wantFirstID {
					t.Errorf("First binding InputID = %v, want %v", first.InputID, tt.wantFirstID)
				}
				if tt.wantFirstMode != "" && first.Mode != tt.wantFirstMode {
					t.Errorf("First binding Mode = %v, want %v", first.Mode, tt.wantFirstMode)
				}
			}
		})
	}
}

// TestParseProfileFileNotFound tests error handling for missing files
func TestParseProfileFileNotFound(t *testing.T) {
	_, err := ParseProfile("/nonexistent/path/profile.xml")
	if err == nil {
		t.Error("ParseProfile() should return error for nonexistent file")
	}
}

// TestParseProfileEmptyPath tests handling of empty path
func TestParseProfileEmptyPath(t *testing.T) {
	bindings, err := ParseProfile("")
	if err != nil {
		t.Errorf("ParseProfile(\"\") should not error, got %v", err)
	}
	if bindings != nil {
		t.Errorf("ParseProfile(\"\") should return nil bindings, got %v", bindings)
	}
}
