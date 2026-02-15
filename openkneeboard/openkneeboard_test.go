package openkneeboard

import (
	"os"
	"path/filepath"
	"simdiag/common"
	"testing"
)

// TestFormatOpenKneeboardAction tests the SCREAMING_SNAKE_CASE to Title Case conversion
func TestFormatOpenKneeboardAction(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single word",
			input: "TOGGLE",
			want:  "Toggle",
		},
		{
			name:  "two words",
			input: "NEXT_TAB",
			want:  "Next Tab",
		},
		{
			name:  "three words",
			input: "PREVIOUS_TAB_SET",
			want:  "Previous Tab Set",
		},
		{
			name:  "four words",
			input: "TOGGLE_FORCE_ZOOM_MODE",
			want:  "Toggle Force Zoom Mode",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single character",
			input: "A",
			want:  "A",
		},
		{
			name:  "trailing underscore",
			input: "NEXT_TAB_",
			want:  "Next Tab",
		},
		{
			name:  "leading underscore",
			input: "_NEXT_TAB",
			want:  "Next Tab",
		},
		{
			name:  "multiple consecutive underscores",
			input: "NEXT__TAB",
			want:  "Next Tab",
		},
		{
			name:  "common OpenKneeboard actions",
			input: "PREVIOUS_PROFILE",
			want:  "Previous Profile",
		},
		{
			name:  "toggle bookmark",
			input: "TOGGLE_BOOKMARK",
			want:  "Toggle Bookmark",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatOpenKneeboardAction(tt.input)
			if got != tt.want {
				t.Errorf("formatOpenKneeboardAction(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseProfiles tests the Profiles.json parsing
func TestParseProfiles(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
		wantEnabled bool
		wantDefault string
		wantActive  string
		wantCount   int
		wantError   bool
	}{
		{
			name: "valid profiles file",
			jsonContent: `{
				"ActiveProfile": "{12345678-1234-1234-1234-123456789012}",
				"DefaultProfile": "{12345678-1234-1234-1234-123456789012}",
				"Enabled": true,
				"LoopProfiles": false,
				"Profiles": [
					{
						"Guid": "{12345678-1234-1234-1234-123456789012}",
						"Name": "Default Profile"
					},
					{
						"Guid": "{87654321-4321-4321-4321-210987654321}",
						"Name": "Custom Profile"
					}
				]
			}`,
			wantEnabled: true,
			wantDefault: "{12345678-1234-1234-1234-123456789012}",
			wantActive:  "{12345678-1234-1234-1234-123456789012}",
			wantCount:   2,
			wantError:   false,
		},
		{
			name: "disabled profiles",
			jsonContent: `{
				"ActiveProfile": "",
				"DefaultProfile": "{12345678-1234-1234-1234-123456789012}",
				"Enabled": false,
				"LoopProfiles": false,
				"Profiles": []
			}`,
			wantEnabled: false,
			wantDefault: "{12345678-1234-1234-1234-123456789012}",
			wantActive:  "",
			wantCount:   0,
			wantError:   false,
		},
		{
			name: "empty profiles array",
			jsonContent: `{
				"ActiveProfile": "{12345678-1234-1234-1234-123456789012}",
				"DefaultProfile": "{12345678-1234-1234-1234-123456789012}",
				"Enabled": true,
				"LoopProfiles": true,
				"Profiles": []
			}`,
			wantEnabled: true,
			wantDefault: "{12345678-1234-1234-1234-123456789012}",
			wantActive:  "{12345678-1234-1234-1234-123456789012}",
			wantCount:   0,
			wantError:   false,
		},
		{
			name:        "invalid JSON",
			jsonContent: `{"ActiveProfile": "invalid json`,
			wantError:   true,
		},
		{
			name:        "empty file",
			jsonContent: ``,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "Profiles.json")
			err := os.WriteFile(tmpFile, []byte(tt.jsonContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Test the function
			profiles, err := ParseProfiles(tmpFile)

			if (err != nil) != tt.wantError {
				t.Errorf("ParseProfiles() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil {
				return // Error was expected
			}

			if profiles.Enabled != tt.wantEnabled {
				t.Errorf("Enabled = %v, want %v", profiles.Enabled, tt.wantEnabled)
			}
			if profiles.DefaultProfile != tt.wantDefault {
				t.Errorf("DefaultProfile = %v, want %v", profiles.DefaultProfile, tt.wantDefault)
			}
			if profiles.ActiveProfile != tt.wantActive {
				t.Errorf("ActiveProfile = %v, want %v", profiles.ActiveProfile, tt.wantActive)
			}
			if len(profiles.Profiles) != tt.wantCount {
				t.Errorf("len(Profiles) = %v, want %v", len(profiles.Profiles), tt.wantCount)
			}
		})
	}
}

// TestParseDirectInput tests the DirectInput.json parsing
func TestParseDirectInput(t *testing.T) {
	tests := []struct {
		name            string
		jsonContent     string
		wantDeviceCount int
		wantFirstGUID   string
		wantFirstName   string
		wantBindings    int
		wantError       bool
	}{
		{
			name: "valid DirectInput with button bindings",
			jsonContent: `{
				"Devices": {
					"{EE6F1C30-3F2E-11F0-8001-444553540000}": {
						"ID": "{EE6F1C30-3F2E-11F0-8001-444553540000}",
						"Kind": "Joystick",
						"Name": "Joystick - HOTAS Warthog",
						"ButtonBindings": [
							{
								"Action": "NEXT_TAB",
								"Buttons": [0, 1]
							},
							{
								"Action": "PREVIOUS_TAB",
								"Buttons": [2]
							}
						]
					}
				}
			}`,
			wantDeviceCount: 1,
			wantFirstGUID:   "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			wantFirstName:   "Joystick - HOTAS Warthog",
			wantBindings:    2,
			wantError:       false,
		},
		{
			name: "multiple devices",
			jsonContent: `{
				"Devices": {
					"{EE6F1C30-3F2E-11F0-8001-444553540000}": {
						"ID": "{EE6F1C30-3F2E-11F0-8001-444553540000}",
						"Kind": "Joystick",
						"Name": "Joystick - HOTAS Warthog",
						"ButtonBindings": []
					},
					"{A7C91C00-3F30-11F0-8001-444553540000}": {
						"ID": "{A7C91C00-3F30-11F0-8001-444553540000}",
						"Kind": "Throttle",
						"Name": "Throttle - HOTAS Warthog",
						"ButtonBindings": []
					}
				}
			}`,
			wantDeviceCount: 2,
			wantError:       false,
		},
		{
			name: "device without bindings",
			jsonContent: `{
				"Devices": {
					"{EE6F1C30-3F2E-11F0-8001-444553540000}": {
						"ID": "{EE6F1C30-3F2E-11F0-8001-444553540000}",
						"Kind": "Joystick",
						"Name": "Test Device"
					}
				}
			}`,
			wantDeviceCount: 1,
			wantFirstGUID:   "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			wantFirstName:   "Test Device",
			wantBindings:    0,
			wantError:       false,
		},
		{
			name: "empty devices",
			jsonContent: `{
				"Devices": {}
			}`,
			wantDeviceCount: 0,
			wantError:       false,
		},
		{
			name:        "invalid JSON",
			jsonContent: `{"Devices": {`,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "DirectInput.json")
			err := os.WriteFile(tmpFile, []byte(tt.jsonContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Test the function
			directInput, err := ParseDirectInput(tmpFile)

			if (err != nil) != tt.wantError {
				t.Errorf("ParseDirectInput() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil {
				return // Error was expected
			}

			if len(directInput.Devices) != tt.wantDeviceCount {
				t.Errorf("len(Devices) = %v, want %v", len(directInput.Devices), tt.wantDeviceCount)
			}

			if tt.wantFirstGUID != "" {
				device, exists := directInput.Devices[tt.wantFirstGUID]
				if !exists {
					t.Errorf("Expected device with GUID %q not found", tt.wantFirstGUID)
					return
				}

				if device.Name != tt.wantFirstName {
					t.Errorf("Device.Name = %v, want %v", device.Name, tt.wantFirstName)
				}

				if len(device.ButtonBindings) != tt.wantBindings {
					t.Errorf("len(ButtonBindings) = %v, want %v", len(device.ButtonBindings), tt.wantBindings)
				}
			}
		})
	}
}

// TestLoadBindings tests the complete binding loading process
func TestLoadBindings(t *testing.T) {
	tests := []struct {
		name               string
		profilesJSON       string
		directInputJSON    string
		deviceGUID         string
		wantBindingCount   int
		wantFirstAction    string
		wantFirstInputID   string
		wantFirstInputType common.InputType
	}{
		{
			name: "load bindings for matching device",
			profilesJSON: `{
				"ActiveProfile": "{PROFILE-GUID}",
				"DefaultProfile": "{PROFILE-GUID}",
				"Enabled": true,
				"LoopProfiles": false,
				"Profiles": [
					{
						"Guid": "{PROFILE-GUID}",
						"Name": "Default"
					}
				]
			}`,
			directInputJSON: `{
				"Devices": {
					"{EE6F1C30-3F2E-11F0-8001-444553540000}": {
						"ID": "{EE6F1C30-3F2E-11F0-8001-444553540000}",
						"Kind": "Joystick",
						"Name": "Test Joystick",
						"ButtonBindings": [
							{
								"Action": "NEXT_TAB",
								"Buttons": [0, 1]
							},
							{
								"Action": "PREVIOUS_TAB",
								"Buttons": [2]
							}
						]
					}
				}
			}`,
			deviceGUID:         "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			wantBindingCount:   3, // 2 buttons for NEXT_TAB + 1 button for PREVIOUS_TAB
			wantFirstAction:    "NEXT_TAB",
			wantFirstInputID:   "1", // Button 0 + 1 offset
			wantFirstInputType: common.Button,
		},
		{
			name: "no bindings for non-matching device",
			profilesJSON: `{
				"ActiveProfile": "{PROFILE-GUID}",
				"DefaultProfile": "{PROFILE-GUID}",
				"Enabled": true,
				"LoopProfiles": false,
				"Profiles": []
			}`,
			directInputJSON: `{
				"Devices": {
					"{DIFFERENT-GUID}": {
						"ID": "{DIFFERENT-GUID}",
						"Kind": "Joystick",
						"Name": "Other Device",
						"ButtonBindings": [
							{
								"Action": "NEXT_TAB",
								"Buttons": [0]
							}
						]
					}
				}
			}`,
			deviceGUID:       "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			wantBindingCount: 0,
		},
		{
			name:             "empty profiles path",
			profilesJSON:     "",
			directInputJSON:  "",
			deviceGUID:       "{ANY-GUID}",
			wantBindingCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.profilesJSON == "" {
				// Test with empty profiles path
				bindings := LoadBindings("", tt.deviceGUID)
				if len(bindings) != tt.wantBindingCount {
					t.Errorf("LoadBindings() returned %d bindings, want %d", len(bindings), tt.wantBindingCount)
				}
				return
			}

			// Create temporary directory structure
			tmpDir := t.TempDir()
			profilesFile := filepath.Join(tmpDir, "Profiles.json")
			profilesDir := filepath.Join(tmpDir, "Profiles", "PROFILE-GUID")
			directInputFile := filepath.Join(profilesDir, "DirectInput.json")

			// Write profiles file
			err := os.WriteFile(profilesFile, []byte(tt.profilesJSON), 0644)
			if err != nil {
				t.Fatalf("Failed to create Profiles.json: %v", err)
			}

			// Create profiles directory and write DirectInput file
			err = os.MkdirAll(profilesDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create profiles directory: %v", err)
			}

			err = os.WriteFile(directInputFile, []byte(tt.directInputJSON), 0644)
			if err != nil {
				t.Fatalf("Failed to create DirectInput.json: %v", err)
			}

			// Test the function
			bindings := LoadBindings(profilesFile, tt.deviceGUID)

			if len(bindings) != tt.wantBindingCount {
				t.Errorf("LoadBindings() returned %d bindings, want %d", len(bindings), tt.wantBindingCount)
			}

			if tt.wantBindingCount > 0 && len(bindings) > 0 {
				first := bindings[0]
				if first.Action != tt.wantFirstAction {
					t.Errorf("First binding Action = %v, want %v", first.Action, tt.wantFirstAction)
				}
				if first.InputID != tt.wantFirstInputID {
					t.Errorf("First binding InputID = %v, want %v", first.InputID, tt.wantFirstInputID)
				}
				if first.InputType != tt.wantFirstInputType {
					t.Errorf("First binding InputType = %v, want %v", first.InputType, tt.wantFirstInputType)
				}
			}
		})
	}
}
