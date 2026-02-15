package il2

import (
	"os"
	"path/filepath"
	"regexp"
	"simdiag/common"
	"testing"
)

// testRegexPatternMatch is a helper function to test regex pattern matching with expected capture groups
func testRegexPatternMatch(t *testing.T, pattern *regexp.Regexp, input string, wantMatch bool, wantGroups []string) {
	t.Helper()
	matches := pattern.FindStringSubmatch(input)
	gotMatch := len(matches) > 0

	if gotMatch != wantMatch {
		t.Errorf("pattern.FindStringSubmatch(%q) matched = %v, want %v", input, gotMatch, wantMatch)
		return
	}

	if wantMatch && len(wantGroups) > 0 {
		expectedLen := len(wantGroups) + 1 // +1 for full match
		if len(matches) != expectedLen {
			t.Errorf("expected %d match groups, got %d", expectedLen, len(matches))
			return
		}
		for i, wantGroup := range wantGroups {
			if matches[i+1] != wantGroup {
				t.Errorf("group[%d] = %v, want %v", i, matches[i+1], wantGroup)
			}
		}
	}
}

// TestFindDeviceGUIDByConfigID tests the device GUID lookup by config ID
func TestFindDeviceGUIDByConfigID(t *testing.T) {
	tests := []struct {
		name     string
		devices  map[string]*common.Device
		configID string
		wantGUID string
	}{
		{
			name: "single device found",
			devices: map[string]*common.Device{
				"1:{GUID-1}": {GUID: "{GUID-1}", Name: "Device 1"},
			},
			configID: "1",
			wantGUID: "{GUID-1}",
		},
		{
			name: "multiple devices, correct one found",
			devices: map[string]*common.Device{
				"1:{GUID-1}": {GUID: "{GUID-1}", Name: "Device 1"},
				"2:{GUID-2}": {GUID: "{GUID-2}", Name: "Device 2"},
				"3:{GUID-3}": {GUID: "{GUID-3}", Name: "Device 3"},
			},
			configID: "2",
			wantGUID: "{GUID-2}",
		},
		{
			name: "device not found",
			devices: map[string]*common.Device{
				"1:{GUID-1}": {GUID: "{GUID-1}", Name: "Device 1"},
			},
			configID: "999",
			wantGUID: "",
		},
		{
			name:     "empty devices map",
			devices:  map[string]*common.Device{},
			configID: "1",
			wantGUID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findDeviceGUIDByConfigID(tt.devices, tt.configID)
			if got != tt.wantGUID {
				t.Errorf("findDeviceGUIDByConfigID() = %v, want %v", got, tt.wantGUID)
			}
		})
	}
}

// TestIL2ButtonPattern tests the IL-2 button pattern regex
func TestIL2ButtonPattern(t *testing.T) {
	buttonPattern := regexp.MustCompile(`^joy(\d+)_b(\d+)$`)

	tests := []struct {
		name         string
		input        string
		wantMatch    bool
		wantConfigID string // Device config ID
		wantButtonID string // Button number
	}{
		{
			name:         "button 1 on device 1",
			input:        "joy1_b1",
			wantMatch:    true,
			wantConfigID: "1",
			wantButtonID: "1",
		},
		{
			name:         "button 25 on device 2",
			input:        "joy2_b25",
			wantMatch:    true,
			wantConfigID: "2",
			wantButtonID: "25",
		},
		{
			name:         "button 105 on device 3",
			input:        "joy3_b105",
			wantMatch:    true,
			wantConfigID: "3",
			wantButtonID: "105",
		},
		{
			name:      "invalid - missing b prefix",
			input:     "joy1_25",
			wantMatch: false,
		},
		{
			name:      "invalid - missing joy prefix",
			input:     "1_b25",
			wantMatch: false,
		},
		{
			name:      "invalid - axis instead of button",
			input:     "joy1_axis_x",
			wantMatch: false,
		},
		{
			name:      "invalid - empty string",
			input:     "",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wantGroups []string
			if tt.wantMatch {
				wantGroups = []string{tt.wantConfigID, tt.wantButtonID}
			}
			testRegexPatternMatch(t, buttonPattern, tt.input, tt.wantMatch, wantGroups)
		})
	}
}

// TestIL2AxisPattern tests the IL-2 axis pattern regex
func TestIL2AxisPattern(t *testing.T) {
	axisPattern := regexp.MustCompile(`^joy(\d+)_axis_([xyzwqsturp])$`)

	tests := []struct {
		name         string
		input        string
		wantMatch    bool
		wantConfigID string
		wantAxis     string
	}{
		{
			name:         "axis X on device 1",
			input:        "joy1_axis_x",
			wantMatch:    true,
			wantConfigID: "1",
			wantAxis:     "x",
		},
		{
			name:         "axis Y on device 2",
			input:        "joy2_axis_y",
			wantMatch:    true,
			wantConfigID: "2",
			wantAxis:     "y",
		},
		{
			name:         "axis Z on device 1",
			input:        "joy1_axis_z",
			wantMatch:    true,
			wantConfigID: "1",
			wantAxis:     "z",
		},
		{
			name:         "axis W on device 3",
			input:        "joy3_axis_w",
			wantMatch:    true,
			wantConfigID: "3",
			wantAxis:     "w",
		},
		{
			name:         "axis Q (rotation X)",
			input:        "joy1_axis_q",
			wantMatch:    true,
			wantConfigID: "1",
			wantAxis:     "q",
		},
		{
			name:         "axis S (rotation Y)",
			input:        "joy1_axis_s",
			wantMatch:    true,
			wantConfigID: "1",
			wantAxis:     "s",
		},
		{
			name:         "axis R (rotation Z)",
			input:        "joy1_axis_r",
			wantMatch:    true,
			wantConfigID: "1",
			wantAxis:     "r",
		},
		{
			name:      "invalid - uppercase axis",
			input:     "joy1_axis_X",
			wantMatch: false,
		},
		{
			name:      "invalid - unsupported axis",
			input:     "joy1_axis_a",
			wantMatch: false,
		},
		{
			name:      "invalid - button instead of axis",
			input:     "joy1_b1",
			wantMatch: false,
		},
		{
			name:      "invalid - empty string",
			input:     "",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wantGroups []string
			if tt.wantMatch {
				wantGroups = []string{tt.wantConfigID, tt.wantAxis}
			}
			testRegexPatternMatch(t, axisPattern, tt.input, tt.wantMatch, wantGroups)
		})
	}
}

// TestIL2POVPattern tests the IL-2 POV pattern regex
func TestIL2POVPattern(t *testing.T) {
	povPattern := regexp.MustCompile(`^joy(\d+)_pov(\d+)_(\d+)$`)

	tests := []struct {
		name          string
		input         string
		wantMatch     bool
		wantConfigID  string
		wantPOVNum    string
		wantDirection string // 0=up, 90=right, 180=down, 270=left
	}{
		{
			name:          "POV 1 up on device 1",
			input:         "joy1_pov1_0",
			wantMatch:     true,
			wantConfigID:  "1",
			wantPOVNum:    "1",
			wantDirection: "0",
		},
		{
			name:          "POV 1 right on device 1",
			input:         "joy1_pov1_90",
			wantMatch:     true,
			wantConfigID:  "1",
			wantPOVNum:    "1",
			wantDirection: "90",
		},
		{
			name:          "POV 1 down on device 2",
			input:         "joy2_pov1_180",
			wantMatch:     true,
			wantConfigID:  "2",
			wantPOVNum:    "1",
			wantDirection: "180",
		},
		{
			name:          "POV 1 left on device 1",
			input:         "joy1_pov1_270",
			wantMatch:     true,
			wantConfigID:  "1",
			wantPOVNum:    "1",
			wantDirection: "270",
		},
		{
			name:          "POV 2 up on device 3",
			input:         "joy3_pov2_0",
			wantMatch:     true,
			wantConfigID:  "3",
			wantPOVNum:    "2",
			wantDirection: "0",
		},
		{
			name:      "invalid - missing angle",
			input:     "joy1_pov1",
			wantMatch: false,
		},
		{
			name:      "invalid - missing pov number",
			input:     "joy1_pov_0",
			wantMatch: false,
		},
		{
			name:      "invalid - button instead of pov",
			input:     "joy1_b1",
			wantMatch: false,
		},
		{
			name:      "invalid - empty string",
			input:     "",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := povPattern.FindStringSubmatch(tt.input)
			gotMatch := len(matches) > 0

			if gotMatch != tt.wantMatch {
				t.Errorf("povPattern.FindStringSubmatch(%q) matched = %v, want %v", tt.input, gotMatch, tt.wantMatch)
				return
			}

			if tt.wantMatch && len(matches) == 4 {
				if matches[1] != tt.wantConfigID {
					t.Errorf("configID = %v, want %v", matches[1], tt.wantConfigID)
				}
				if matches[2] != tt.wantPOVNum {
					t.Errorf("povNum = %v, want %v", matches[2], tt.wantPOVNum)
				}
				if matches[3] != tt.wantDirection {
					t.Errorf("direction = %v, want %v", matches[3], tt.wantDirection)
				}
			}
		})
	}
}

// TestParseIL2Devices tests the devices.txt parsing
func TestParseIL2Devices(t *testing.T) {
	tests := []struct {
		name            string
		fileContent     string
		wantDeviceCount int
		wantFirstKey    string
		wantFirstDevice *common.Device
		wantError       bool
	}{
		{
			name: "single device",
			fileContent: `configId,guid,model|
1,%22b0c891c0-3f30-11f0-0000545345440380%22,T-Rudder|`,
			wantDeviceCount: 1,
			wantFirstKey:    "1:b0c891c0-3f30-11f0-0000545345440380",
			wantFirstDevice: &common.Device{
				GUID: "b0c891c0-3f30-11f0-0000545345440380",
				Name: "T-Rudder",
			},
			wantError: false,
		},
		{
			name: "multiple devices",
			fileContent: `configId,guid,model|
1,%22b0c891c0-3f30-11f0-0000545345440380%22,T-Rudder|
2,%22ee6f1c30-3f2e-11f0-8001-444553540000%22,Joystick+-+HOTAS+Warthog|
3,%22a7c91c00-3f30-11f0-8001-444553540000%22,Throttle+-+HOTAS+Warthog|`,
			wantDeviceCount: 3,
			wantError:       false,
		},
		{
			name: "with comments and empty lines",
			fileContent: `// Comment line
configId,guid,model|

1,%22b0c891c0-3f30-11f0-0000545345440380%22,T-Rudder|
// Another comment
2,%22ee6f1c30-3f2e-11f0-8001-444553540000%22,Joystick|`,
			wantDeviceCount: 2,
			wantError:       false,
		},
		{
			name: "device with special characters in name",
			fileContent: `configId,guid,model|
1,%22b0c891c0-3f30-11f0-0000545345440380%22,Device+%26+Controller|`,
			wantDeviceCount: 1,
			wantFirstKey:    "1:b0c891c0-3f30-11f0-0000545345440380",
			wantFirstDevice: &common.Device{
				GUID: "b0c891c0-3f30-11f0-0000545345440380",
				Name: "Device & Controller",
			},
			wantError: false,
		},
		{
			name:        "empty file",
			fileContent: `configId,guid,model|`,
			wantError:   true, // No devices found
		},
		{
			name:        "only comments",
			fileContent: `// Just a comment`,
			wantError:   true, // No devices found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "devices.txt")
			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Test the function
			devices, err := parseIL2Devices(tmpFile)

			if (err != nil) != tt.wantError {
				t.Errorf("parseIL2Devices() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil {
				return // Error was expected
			}

			if len(devices) != tt.wantDeviceCount {
				t.Errorf("parseIL2Devices() returned %d devices, want %d", len(devices), tt.wantDeviceCount)
			}

			if tt.wantFirstDevice != nil && tt.wantFirstKey != "" {
				device, exists := devices[tt.wantFirstKey]
				if !exists {
					t.Errorf("Expected device with key %q not found", tt.wantFirstKey)
					return
				}

				if device.GUID != tt.wantFirstDevice.GUID {
					t.Errorf("device.GUID = %v, want %v", device.GUID, tt.wantFirstDevice.GUID)
				}
				if device.Name != tt.wantFirstDevice.Name {
					t.Errorf("device.Name = %v, want %v", device.Name, tt.wantFirstDevice.Name)
				}
			}
		})
	}
}
