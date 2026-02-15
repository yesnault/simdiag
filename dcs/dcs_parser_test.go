package dcs

import (
	"simdiag/common"
	"testing"
)

// TestExtractBalancedBraces tests the brace-counting algorithm for nested Lua tables
func TestExtractBalancedBraces(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		startPos  int
		wantTable string
		wantEnd   int
	}{
		{
			name:      "simple table",
			content:   `local x = {a=1, b=2} more text`,
			startPos:  10,
			wantTable: `a=1, b=2`,
			wantEnd:   20,
		},
		{
			name:      "nested table",
			content:   `local x = {outer={inner=1}} more`,
			startPos:  10,
			wantTable: `outer={inner=1}`,
			wantEnd:   27,
		},
		{
			name:      "deeply nested table (5 levels)",
			content:   `local x = {a={b={c={d={e=1}}}}} more`,
			startPos:  10,
			wantTable: `a={b={c={d={e=1}}}}`,
			wantEnd:   31,
		},
		{
			name:      "axis table with multiple nested levels",
			content:   `["a2001cdnil"] = {["name"]="Roulis",["removed"]={[1]={["key"]="JOY_X"}},["added"]={[1]={["key"]="JOY_X",["filter"]={["curvature"]={[1]=0},["deadzone"]=0.03}}}}`,
			startPos:  17,
			wantTable: `["name"]="Roulis",["removed"]={[1]={["key"]="JOY_X"}},["added"]={[1]={["key"]="JOY_X",["filter"]={["curvature"]={[1]=0},["deadzone"]=0.03}}}`,
			wantEnd:   159,
		},
		{
			name:      "empty table",
			content:   `local x = {} more`,
			startPos:  10,
			wantTable: ``,
			wantEnd:   12,
		},
		{
			name:      "table with string containing braces",
			content:   `local x = {text="{}",val=1} more`,
			startPos:  10,
			wantTable: `text="{}",val=1`,
			wantEnd:   27,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTable, gotEnd := extractBalancedBraces(tt.content, tt.startPos)
			if gotTable != tt.wantTable {
				t.Errorf("extractBalancedBraces() gotTable = %v, want %v", gotTable, tt.wantTable)
			}
			if gotEnd != tt.wantEnd {
				t.Errorf("extractBalancedBraces() gotEnd = %v, want %v", gotEnd, tt.wantEnd)
			}
		})
	}
}

// TestParseDCSJoyInput tests the joystick input parsing logic
func TestParseDCSJoyInput(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantInputType common.InputType
		wantInputID   string
	}{
		// Button tests
		{
			name:          "button 1",
			input:         "JOY_BTN1",
			wantInputType: common.Button,
			wantInputID:   "1",
		},
		{
			name:          "button 25",
			input:         "JOY_BTN25",
			wantInputType: common.Button,
			wantInputID:   "25",
		},
		{
			name:          "button with _OFF suffix",
			input:         "JOY_BTN24_OFF",
			wantInputType: common.Button,
			wantInputID:   "24_OFF",
		},
		// Axis tests
		{
			name:          "axis X",
			input:         "JOY_X",
			wantInputType: common.Axis,
			wantInputID:   "X",
		},
		{
			name:          "axis Y",
			input:         "JOY_Y",
			wantInputType: common.Axis,
			wantInputID:   "Y",
		},
		{
			name:          "axis Z",
			input:         "JOY_Z",
			wantInputType: common.Axis,
			wantInputID:   "Z",
		},
		{
			name:          "axis RX",
			input:         "JOY_RX",
			wantInputType: common.Axis,
			wantInputID:   "RX",
		},
		{
			name:          "axis RY",
			input:         "JOY_RY",
			wantInputType: common.Axis,
			wantInputID:   "RY",
		},
		{
			name:          "axis RZ",
			input:         "JOY_RZ",
			wantInputType: common.Axis,
			wantInputID:   "RZ",
		},
		{
			name:          "axis SLIDER",
			input:         "JOY_SLIDER1",
			wantInputType: common.Axis,
			wantInputID:   "SLIDER_1",
		},
		{
			name:          "axis SLIDER2",
			input:         "JOY_SLIDER2",
			wantInputType: common.Axis,
			wantInputID:   "SLIDER_2",
		},
		// POV/Hat tests
		{
			name:          "POV 1 up",
			input:         "JOY_BTN_POV1_U",
			wantInputType: common.Hat,
			wantInputID:   "1_U",
		},
		{
			name:          "POV 1 down",
			input:         "JOY_BTN_POV1_D",
			wantInputType: common.Hat,
			wantInputID:   "1_D",
		},
		{
			name:          "POV 1 left",
			input:         "JOY_BTN_POV1_L",
			wantInputType: common.Hat,
			wantInputID:   "1_L",
		},
		{
			name:          "POV 1 right",
			input:         "JOY_BTN_POV1_R",
			wantInputType: common.Hat,
			wantInputID:   "1_R",
		},
		{
			name:          "POV 2 up",
			input:         "JOY_BTN_POV2_U",
			wantInputType: common.Hat,
			wantInputID:   "2_U",
		},
		// Unknown inputs default to button
		{
			name:          "unknown input",
			input:         "JOY_UNKNOWN_INPUT",
			wantInputType: common.Button,
			wantInputID:   "UNKNOWN_INPUT",
		},
		{
			name:          "empty input",
			input:         "JOY_",
			wantInputType: common.Button,
			wantInputID:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInputType, gotInputID := parseDCSJoyInput(tt.input)
			if gotInputType != tt.wantInputType {
				t.Errorf("parseDCSJoyInput() gotInputType = %v, want %v", gotInputType, tt.wantInputType)
			}
			if gotInputID != tt.wantInputID {
				t.Errorf("parseDCSJoyInput() gotInputID = %v, want %v", gotInputID, tt.wantInputID)
			}
		})
	}
}

// TestParseKeyDiffs tests the key binding extraction from Lua content
func TestParseKeyDiffs(t *testing.T) {
	tests := []struct {
		name             string
		fileContent      string
		guid             string
		deviceName       string
		wantCount        int
		wantFirst        *common.Binding
		wantModifierKeys []string // For testing modifier keys in first binding
	}{
		{
			name: "single button binding",
			fileContent: `
				["keyDiffs"] = {
					["d3001pnilu3001cd2vd1vpnilvu0"] = {
						["added"] = {
							[1] = {
								["key"] = "JOY_BTN1",
							},
						},
						["name"] = "Master Arm",
					},
				}`,
			guid:       "{TEST-GUID}",
			deviceName: "Test Device",
			wantCount:  1,
			wantFirst: &common.Binding{
				InputType:  common.Button,
				InputID:    "1",
				Action:     "Master Arm",
				DeviceGUID: "{TEST-GUID}",
				DeviceName: "Test Device",
			},
		},
		{
			name: "button binding with modifier",
			fileContent: `
				["keyDiffs"] = {
					["d3002pnilu3002cd2vd1vpnilvu0"] = {
						["added"] = {
							[1] = {
								["key"] = "JOY_BTN2",
								["reformers"] = {
									[1] = "RShift",
								},
							},
						},
						["name"] = "Gun Trigger",
					},
				}`,
			guid:             "{TEST-GUID}",
			deviceName:       "Test Device",
			wantCount:        1,
			wantModifierKeys: []string{"RShift"},
			wantFirst: &common.Binding{
				InputType:  common.Button,
				InputID:    "2",
				Action:     "Gun Trigger",
				DeviceGUID: "{TEST-GUID}",
				DeviceName: "Test Device",
			},
		},
		{
			name: "POV binding",
			fileContent: `
				["keyDiffs"] = {
					["d3003pnilu3003cd2vd1vpnilvu0"] = {
						["added"] = {
							[1] = {
								["key"] = "JOY_BTN_POV1_U",
							},
						},
						["name"] = "Trim Up",
					},
				}`,
			guid:       "{TEST-GUID}",
			deviceName: "Test Device",
			wantCount:  1,
			wantFirst: &common.Binding{
				InputType:  common.Hat,
				InputID:    "1_U",
				Action:     "Trim Up",
				DeviceGUID: "{TEST-GUID}",
				DeviceName: "Test Device",
			},
		},
		{
			name: "multiple bindings",
			fileContent: `local diff = {
				["keyDiffs"] = {
					["d3001pnilu3001cd2vd1vpnilvu0"] = {
						["added"] = {
							[1] = {
								["key"] = "JOY_BTN1",
							},
						},
						["name"] = "Action 1",
					},
					["d3002pnilu3002cd2vd1vpnilvu0"] = {
						["added"] = {
							[1] = {
								["key"] = "JOY_BTN2",
							},
						},
						["name"] = "Action 2",
					},
				}
			}
			return diff`,
			guid:       "{TEST-GUID}",
			deviceName: "Test Device",
			wantCount:  2,
			wantFirst:  nil, // Don't check first binding for multiple bindings
		},
		{
			name:        "empty keyDiffs",
			fileContent: `["keyDiffs"] = {}`,
			guid:        "{TEST-GUID}",
			deviceName:  "Test Device",
			wantCount:   0,
			wantFirst:   nil,
		},
		{
			name:        "no keyDiffs section",
			fileContent: `["other"] = {}`,
			guid:        "{TEST-GUID}",
			deviceName:  "Test Device",
			wantCount:   0,
			wantFirst:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &common.Profile{
				Devices:  make(map[string]*common.Device),
				Bindings: make([]common.Binding, 0),
			}

			bindings := parseKeyDiffs(tt.fileContent, tt.guid, tt.deviceName, profile)

			if len(bindings) != tt.wantCount {
				t.Errorf("parseKeyDiffs() returned %d bindings, want %d", len(bindings), tt.wantCount)
			}

			if tt.wantFirst != nil && len(bindings) > 0 {
				got := bindings[0]
				if got.InputType != tt.wantFirst.InputType {
					t.Errorf("binding.InputType = %v, want %v", got.InputType, tt.wantFirst.InputType)
				}
				if got.InputID != tt.wantFirst.InputID {
					t.Errorf("binding.InputID = %v, want %v", got.InputID, tt.wantFirst.InputID)
				}
				if got.Action != tt.wantFirst.Action {
					t.Errorf("binding.Action = %v, want %v", got.Action, tt.wantFirst.Action)
				}
				if got.DeviceGUID != tt.wantFirst.DeviceGUID {
					t.Errorf("binding.DeviceGUID = %v, want %v", got.DeviceGUID, tt.wantFirst.DeviceGUID)
				}
				if got.DeviceName != tt.wantFirst.DeviceName {
					t.Errorf("binding.DeviceName = %v, want %v", got.DeviceName, tt.wantFirst.DeviceName)
				}

				// Check modifiers if specified
				if tt.wantModifierKeys != nil {
					if len(got.Modifiers) != len(tt.wantModifierKeys) {
						t.Errorf("binding has %d modifiers, want %d", len(got.Modifiers), len(tt.wantModifierKeys))
					} else if len(tt.wantModifierKeys) > 0 && len(got.Modifiers) > 0 {
						// Check first modifier's first key
						if len(got.Modifiers[0].Keys) == 0 || got.Modifiers[0].Keys[0] != tt.wantModifierKeys[0] {
							gotKey := ""
							if len(got.Modifiers) > 0 && len(got.Modifiers[0].Keys) > 0 {
								gotKey = got.Modifiers[0].Keys[0]
							}
							t.Errorf("binding.Modifiers[0].Keys[0] = %v, want %v", gotKey, tt.wantModifierKeys[0])
						}
					}
				}
			}
		})
	}
}

// TestParseAxisDiffs tests the axis binding extraction from Lua content
func TestParseAxisDiffs(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		guid        string
		deviceName  string
		wantCount   int
		wantFirst   *common.Binding
	}{
		{
			name: "single axis binding",
			fileContent: `
				["axisDiffs"] = {
					["a2001cdnil"] = {
						["name"] = "Pitch",
						["added"] = {
							[1] = {
								["key"] = "JOY_Y",
							},
						},
					},
				}`,
			guid:       "{TEST-GUID}",
			deviceName: "Test Joystick",
			wantCount:  1,
			wantFirst: &common.Binding{
				InputType:  common.Axis,
				InputID:    "Y",
				Action:     "Pitch",
				DeviceGUID: "{TEST-GUID}",
				DeviceName: "Test Joystick",
			},
		},
		{
			name: "axis X binding",
			fileContent: `
				["axisDiffs"] = {
					["a2002cdnil"] = {
						["name"] = "Roll",
						["added"] = {
							[1] = {
								["key"] = "JOY_X",
							},
						},
					},
				}`,
			guid:       "{TEST-GUID}",
			deviceName: "Test Joystick",
			wantCount:  1,
			wantFirst: &common.Binding{
				InputType:  common.Axis,
				InputID:    "X",
				Action:     "Roll",
				DeviceGUID: "{TEST-GUID}",
				DeviceName: "Test Joystick",
			},
		},
		{
			name: "slider binding",
			fileContent: `
				["axisDiffs"] = {
					["a2003cdnil"] = {
						["name"] = "Throttle",
						["added"] = {
							[1] = {
								["key"] = "JOY_SLIDER1",
							},
						},
					},
				}`,
			guid:       "{TEST-GUID}",
			deviceName: "Test Throttle",
			wantCount:  1,
			wantFirst: &common.Binding{
				InputType:  common.Axis,
				InputID:    "SLIDER_1",
				Action:     "Throttle",
				DeviceGUID: "{TEST-GUID}",
				DeviceName: "Test Throttle",
			},
		},
		{
			name: "deeply nested axis binding",
			fileContent: `
				["axisDiffs"] = {
					["a2001cdnil"] = {
						["name"] = "Roulis",
						["removed"] = {
							[1] = {
								["key"] = "JOY_X",
							},
						},
						["added"] = {
							[1] = {
								["key"] = "JOY_X",
								["filter"] = {
									["curvature"] = {
										[1] = 0,
									},
									["deadzone"] = 0.03,
									["hardwareDetent"] = false,
									["hardwareDetentAB"] = 0,
									["hardwareDetentMax"] = 0,
									["invert"] = false,
									["saturationX"] = 1,
									["saturationY"] = 1,
									["slider"] = false,
								},
							},
						},
					},
				}`,
			guid:       "{TEST-GUID}",
			deviceName: "Test Joystick",
			wantCount:  1,
			wantFirst: &common.Binding{
				InputType:  common.Axis,
				InputID:    "X",
				Action:     "Roulis",
				DeviceGUID: "{TEST-GUID}",
				DeviceName: "Test Joystick",
			},
		},
		{
			name:        "empty axisDiffs",
			fileContent: `["axisDiffs"] = {}`,
			guid:        "{TEST-GUID}",
			deviceName:  "Test Device",
			wantCount:   0,
			wantFirst:   nil,
		},
		{
			name:        "no axisDiffs section",
			fileContent: `["other"] = {}`,
			guid:        "{TEST-GUID}",
			deviceName:  "Test Device",
			wantCount:   0,
			wantFirst:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &common.Profile{
				Devices:  make(map[string]*common.Device),
				Bindings: make([]common.Binding, 0),
			}

			parseAxisDiffs(tt.fileContent, tt.guid, tt.deviceName, profile)
			bindings := profile.Bindings

			if len(bindings) != tt.wantCount {
				t.Errorf("parseAxisDiffs() created %d bindings, want %d", len(bindings), tt.wantCount)
			}

			if tt.wantFirst != nil && len(bindings) > 0 {
				got := bindings[0]
				if got.InputType != tt.wantFirst.InputType {
					t.Errorf("binding.InputType = %v, want %v", got.InputType, tt.wantFirst.InputType)
				}
				if got.InputID != tt.wantFirst.InputID {
					t.Errorf("binding.InputID = %v, want %v", got.InputID, tt.wantFirst.InputID)
				}
				if got.Action != tt.wantFirst.Action {
					t.Errorf("binding.Action = %v, want %v", got.Action, tt.wantFirst.Action)
				}
				if got.DeviceGUID != tt.wantFirst.DeviceGUID {
					t.Errorf("binding.DeviceGUID = %v, want %v", got.DeviceGUID, tt.wantFirst.DeviceGUID)
				}
				if got.DeviceName != tt.wantFirst.DeviceName {
					t.Errorf("binding.DeviceName = %v, want %v", got.DeviceName, tt.wantFirst.DeviceName)
				}
			}
		})
	}
}
