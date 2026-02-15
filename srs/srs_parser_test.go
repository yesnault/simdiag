package srs

import (
	"os"
	"path/filepath"
	"simdiag/common"
	"strings"
	"testing"
)

// TestParseSRSConfig tests the SRS config file parsing
func TestParseSRSConfig(t *testing.T) {
	tests := []struct {
		name             string
		configContent    string
		simType          common.SimulationType
		wantGUIDs        []string
		wantFirstGUID    string
		wantFirstButton  string
		wantFirstAction  string
		wantBindingCount int
		wantError        bool
	}{
		{
			name: "DCS SRS config with single binding",
			configContent: `# SRS Configuration File
[RadioSwitch]
name="Radio Switch"
button=16
guid={EE6F1C30-3F2E-11F0-8001-444553540000}
`,
			simType:          common.DCSWorld,
			wantGUIDs:        []string{"{ee6f1c30-3f2e-11f0-8001-444553540000}"},
			wantFirstGUID:    "{ee6f1c30-3f2e-11f0-8001-444553540000}",
			wantFirstButton:  "17", // button=16 in SRS (0-indexed) becomes 17 (1-indexed)
			wantFirstAction:  "SRS: RadioSwitch",
			wantBindingCount: 1,
			wantError:        false,
		},
		{
			name: "IL-2 SRS config with multiple bindings",
			configContent: `[RadioSwitch]
name="Throttle - HOTAS Warthog"
button=16
guid={A7C91C00-3F30-11F0-8001-444553540000}

[RadioChannelUp]
name="Throttle - HOTAS Warthog"
button=17
guid={A7C91C00-3F30-11F0-8001-444553540000}

[RadioChannelDown]
name="Throttle - HOTAS Warthog"
button=18
guid={A7C91C00-3F30-11F0-8001-444553540000}
`,
			simType:          common.IL2Sturmovik,
			wantGUIDs:        []string{"{a7c91c00-3f30-11f0-8001-444553540000}"},
			wantFirstGUID:    "{a7c91c00-3f30-11f0-8001-444553540000}",
			wantFirstButton:  "17",
			wantFirstAction:  "SRS: RadioSwitch",
			wantBindingCount: 3,
			wantError:        false,
		},
		{
			name: "multiple devices",
			configContent: `[RadioSwitch]
name="Joystick"
button=24
guid={EE6F1C30-3F2E-11F0-8001-444553540000}

[RadioChannelUp]
name="Throttle"
button=16
guid={A7C91C00-3F30-11F0-8001-444553540000}
`,
			simType:          common.DCSWorld,
			wantGUIDs:        []string{"{ee6f1c30-3f2e-11f0-8001-444553540000}", "{a7c91c00-3f30-11f0-8001-444553540000}"},
			wantBindingCount: 2,
			wantError:        false,
		},
		{
			name: "button 0 becomes button 1",
			configContent: `[PTTButton]
name="Test Device"
button=0
guid={TEST-GUID}
`,
			simType:          common.DCSWorld,
			wantFirstGUID:    "{test-guid}",
			wantFirstButton:  "1", // button=0 + 1 offset
			wantBindingCount: 1,
			wantError:        false,
		},
		{
			name: "high button number",
			configContent: `[RadioSwitch]
name="Test Device"
button=105
guid={TEST-GUID}
`,
			simType:          common.DCSWorld,
			wantFirstGUID:    "{test-guid}",
			wantFirstButton:  "106", // button=105 + 1 offset
			wantBindingCount: 1,
			wantError:        false,
		},
		{
			name: "with comments and empty lines",
			configContent: `# This is a comment
; Another comment style

[RadioSwitch]
# Comment in section
name="Test Device"
button=16
guid={TEST-GUID}

# Comment at end
`,
			simType:          common.DCSWorld,
			wantFirstGUID:    "{test-guid}",
			wantBindingCount: 1,
			wantError:        false,
		},
		{
			name: "incomplete section (no button)",
			configContent: `[RadioSwitch]
name="Test Device"
guid={TEST-GUID}
`,
			simType:          common.DCSWorld,
			wantBindingCount: 0,
			wantError:        false,
		},
		{
			name: "incomplete section (no GUID)",
			configContent: `[RadioSwitch]
name="Test Device"
button=16
`,
			simType:          common.DCSWorld,
			wantBindingCount: 0,
			wantError:        false,
		},
		{
			name: "section with quoted name containing spaces",
			configContent: `[RadioSwitch]
name="Throttle - HOTAS Warthog Throttle"
button=16
guid={A7C91C00-3F30-11F0-8001-444553540000}
`,
			simType:          common.DCSWorld,
			wantFirstGUID:    "{a7c91c00-3f30-11f0-8001-444553540000}",
			wantBindingCount: 1,
			wantError:        false,
		},
		{
			name: "empty file",
			configContent: `# Just comments
; No actual config
`,
			simType:          common.DCSWorld,
			wantBindingCount: 0,
			wantError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tmpDir := t.TempDir()
			var configPath string

			if tt.simType == common.DCSWorld {
				// DCS: srsPath/Client/default.cfg
				clientDir := filepath.Join(tmpDir, "Client")
				err := os.MkdirAll(clientDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create Client directory: %v", err)
				}
				configPath = filepath.Join(clientDir, "default.cfg")
			} else {
				// IL-2: srsPath/default.cfg
				configPath = filepath.Join(tmpDir, "default.cfg")
			}

			// Write config file
			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Test the function
			bindings, err := ParseSRSConfig(tmpDir, tt.simType)

			if (err != nil) != tt.wantError {
				t.Errorf("ParseSRSConfig() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil {
				return // Error was expected
			}

			// Check number of devices
			if len(tt.wantGUIDs) > 0 {
				if len(bindings) != len(tt.wantGUIDs) {
					t.Errorf("ParseSRSConfig() returned %d devices, want %d", len(bindings), len(tt.wantGUIDs))
				}

				// Check that all expected GUIDs are present
				for _, guid := range tt.wantGUIDs {
					if _, exists := bindings[guid]; !exists {
						t.Errorf("Expected GUID %q not found in bindings", guid)
					}
				}
			}

			// Check total binding count across all devices
			totalBindings := 0
			for _, deviceBindings := range bindings {
				totalBindings += len(deviceBindings)
			}
			if totalBindings != tt.wantBindingCount {
				t.Errorf("ParseSRSConfig() returned %d total bindings, want %d", totalBindings, tt.wantBindingCount)
			}

			// Check first binding details if specified
			if tt.wantFirstGUID != "" && len(bindings[tt.wantFirstGUID]) > 0 {
				firstBinding := bindings[tt.wantFirstGUID][0]

				if tt.wantFirstButton != "" && firstBinding.InputID != tt.wantFirstButton {
					t.Errorf("First binding InputID = %v, want %v", firstBinding.InputID, tt.wantFirstButton)
				}

				if tt.wantFirstAction != "" && firstBinding.Action != tt.wantFirstAction {
					t.Errorf("First binding Action = %v, want %v", firstBinding.Action, tt.wantFirstAction)
				}

				// All SRS bindings should be buttons
				if firstBinding.InputType != common.Button {
					t.Errorf("First binding InputType = %v, want Button", firstBinding.InputType)
				}

				// Check description format
				if !strings.HasPrefix(firstBinding.Description, "SRS ") {
					t.Errorf("First binding Description = %v, should start with 'SRS '", firstBinding.Description)
				}
			}
		})
	}
}

// TestParseSRSConfigFileNotFound tests error handling for missing files
func TestParseSRSConfigFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Test DCS (expects Client/default.cfg)
	_, err := ParseSRSConfig(tmpDir, common.DCSWorld)
	if err == nil {
		t.Error("ParseSRSConfig() should return error when config file is missing")
	}

	// Test IL-2 (expects default.cfg)
	_, err = ParseSRSConfig(tmpDir, common.IL2Sturmovik)
	if err == nil {
		t.Error("ParseSRSConfig() should return error when config file is missing")
	}
}

// TestParseSRSConfigGUIDNormalization tests that GUIDs are normalized to lowercase
func TestParseSRSConfigGUIDNormalization(t *testing.T) {
	tests := []struct {
		name         string
		inputGUID    string
		expectedGUID string
	}{
		{
			name:         "uppercase GUID",
			inputGUID:    "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			expectedGUID: "{ee6f1c30-3f2e-11f0-8001-444553540000}",
		},
		{
			name:         "mixed case GUID",
			inputGUID:    "{Ee6F1c30-3f2E-11F0-8001-444553540000}",
			expectedGUID: "{ee6f1c30-3f2e-11f0-8001-444553540000}",
		},
		{
			name:         "lowercase GUID",
			inputGUID:    "{ee6f1c30-3f2e-11f0-8001-444553540000}",
			expectedGUID: "{ee6f1c30-3f2e-11f0-8001-444553540000}",
		},
		{
			name:         "GUID with spaces",
			inputGUID:    " {EE6F1C30-3F2E-11F0-8001-444553540000} ",
			expectedGUID: "{ee6f1c30-3f2e-11f0-8001-444553540000}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			clientDir := filepath.Join(tmpDir, "Client")
			err := os.MkdirAll(clientDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}

			configContent := `[RadioSwitch]
name="Test Device"
button=16
guid=` + tt.inputGUID + `
`
			configPath := filepath.Join(clientDir, "default.cfg")
			err = os.WriteFile(configPath, []byte(configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			bindings, err := ParseSRSConfig(tmpDir, common.DCSWorld)
			if err != nil {
				t.Fatalf("ParseSRSConfig() error = %v", err)
			}

			if _, exists := bindings[tt.expectedGUID]; !exists {
				t.Errorf("Expected normalized GUID %q not found. Got GUIDs: %v", tt.expectedGUID, getKeys(bindings))
			}
		})
	}
}

// Helper function to get map keys
func getKeys(m map[string][]common.Binding) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
