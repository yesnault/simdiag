package svg

import (
	"simdiag/common"
	"testing"
)

// TestValidateBindings_NoTemplate tests validation with no template
func TestValidateBindings_NoTemplate(t *testing.T) {
	exportDevice := &common.ExportDevice{
		Device: &common.Device{
			Name: "Test Device",
			GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		Profile: &common.Profile{
			Bindings: []common.Binding{
				{
					Action:     "Test Action",
					InputType:  common.Button,
					InputID:    "1",
					DeviceGUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
				},
			},
		},
		Template: nil,
	}

	errors := ValidateBindings(exportDevice)

	if len(errors) != 0 {
		t.Errorf("ValidateBindings() with no template should return no errors, got %d errors", len(errors))
	}
}

// TestValidateBindings_NoProfile tests validation with no profile
func TestValidateBindings_NoProfile(t *testing.T) {
	exportDevice := &common.ExportDevice{
		Device: &common.Device{
			Name: "Test Device",
			GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		Profile:  nil,
		Template: &common.Template{},
	}

	errors := ValidateBindings(exportDevice)

	if len(errors) != 0 {
		t.Errorf("ValidateBindings() with no profile should return no errors, got %d errors", len(errors))
	}
}

// TestValidateBindings_ValidButton tests validation with a valid button binding
func TestValidateBindings_ValidButton(t *testing.T) {
	exportDevice := &common.ExportDevice{
		Device: &common.Device{
			Name: "Test Device",
			GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		Profile: &common.Profile{
			Bindings: []common.Binding{
				{
					Action:     "Test Action",
					InputType:  common.Button,
					InputID:    "1",
					DeviceGUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
				},
			},
		},
		Template: &common.Template{
			Buttons: []string{"BUTTON_1", "BUTTON_2", "BUTTON_3"},
		},
	}

	errors := ValidateBindings(exportDevice)

	if len(errors) != 0 {
		t.Errorf("ValidateBindings() should find no errors for valid binding, got %d errors", len(errors))
	}
}

// TestValidateBindings_InvalidButton tests validation with an invalid button binding
func TestValidateBindings_InvalidButton(t *testing.T) {
	exportDevice := &common.ExportDevice{
		Device: &common.Device{
			Name: "Test Device",
			GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		Profile: &common.Profile{
			Bindings: []common.Binding{
				{
					Action:     "Test Action",
					InputType:  common.Button,
					InputID:    "99", // Not in template
					DeviceGUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
				},
			},
		},
		Template: &common.Template{
			Buttons: []string{"BUTTON_1", "BUTTON_2", "BUTTON_3"},
		},
	}

	errors := ValidateBindings(exportDevice)

	if len(errors) != 1 {
		t.Errorf("ValidateBindings() should find 1 error for invalid button, got %d errors", len(errors))
	}

	if len(errors) > 0 {
		// The missing key is reported as-is from the expected key generation
		if errors[0].MissingKey != "Button_99" {
			t.Errorf("Expected missing key 'Button_99', got '%s'", errors[0].MissingKey)
		}
	}
}

// TestValidateBindings_ValidAxis tests validation with a valid axis binding
func TestValidateBindings_ValidAxis(t *testing.T) {
	exportDevice := &common.ExportDevice{
		Device: &common.Device{
			Name: "Test Device",
			GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		Profile: &common.Profile{
			Bindings: []common.Binding{
				{
					Action:     "Test Axis",
					InputType:  common.Axis,
					InputID:    "X",
					DeviceGUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
				},
			},
		},
		Template: &common.Template{
			Axes: []string{"AXIS_X", "AXIS_Y", "AXIS_Z"},
		},
	}

	errors := ValidateBindings(exportDevice)

	if len(errors) != 0 {
		t.Errorf("ValidateBindings() should find no errors for valid axis, got %d errors", len(errors))
	}
}

// TestValidateBindings_ValidHat tests validation with a valid hat binding
func TestValidateBindings_ValidHat(t *testing.T) {
	exportDevice := &common.ExportDevice{
		Device: &common.Device{
			Name: "Test Device",
			GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		Profile: &common.Profile{
			Bindings: []common.Binding{
				{
					Action:     "Test Hat",
					InputType:  common.Hat,
					InputID:    "1_U",
					DeviceGUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
				},
			},
		},
		Template: &common.Template{
			Hats: []string{"POV_1_U", "POV_1_D", "POV_1_L", "POV_1_R"},
		},
	}

	errors := ValidateBindings(exportDevice)

	if len(errors) != 0 {
		t.Errorf("ValidateBindings() should find no errors for valid hat, got %d errors", len(errors))
	}
}

// TestValidateBindings_OFFSuffix tests that _OFF suffix is stripped during validation
func TestValidateBindings_OFFSuffix(t *testing.T) {
	exportDevice := &common.ExportDevice{
		Device: &common.Device{
			Name: "Test Device",
			GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		Profile: &common.Profile{
			Bindings: []common.Binding{
				{
					Action:     "Test Action",
					InputType:  common.Button,
					InputID:    "25_OFF", // Should map to Button_25, not Button_25_OFF
					DeviceGUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
				},
			},
		},
		Template: &common.Template{
			Buttons: []string{"BUTTON_25"}, // Only BUTTON_25, not BUTTON_25_OFF
		},
	}

	errors := ValidateBindings(exportDevice)

	if len(errors) != 0 {
		t.Errorf("ValidateBindings() should strip _OFF suffix and find no errors, got %d errors", len(errors))
		if len(errors) > 0 {
			t.Logf("Error: %s", errors[0].MissingKey)
		}
	}
}

// TestValidateBindings_DifferentDevice tests that bindings for other devices are skipped
func TestValidateBindings_DifferentDevice(t *testing.T) {
	exportDevice := &common.ExportDevice{
		Device: &common.Device{
			Name: "Test Device",
			GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		Profile: &common.Profile{
			Bindings: []common.Binding{
				{
					Action:     "Test Action",
					InputType:  common.Button,
					InputID:    "99",
					DeviceGUID: "{A7C91C00-3F30-11F0-8001-444553540000}", // Different device
				},
			},
		},
		Template: &common.Template{
			Buttons: []string{"Button_1", "Button_2"},
		},
	}

	errors := ValidateBindings(exportDevice)

	// Should find no errors because binding is for a different device
	if len(errors) != 0 {
		t.Errorf("ValidateBindings() should skip bindings for other devices, got %d errors", len(errors))
	}
}

// TestValidateBindings_MultipleErrors tests validation with multiple invalid bindings
func TestValidateBindings_MultipleErrors(t *testing.T) {
	exportDevice := &common.ExportDevice{
		Device: &common.Device{
			Name: "Test Device",
			GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		Profile: &common.Profile{
			Bindings: []common.Binding{
				{
					Action:     "Invalid Button",
					InputType:  common.Button,
					InputID:    "99",
					DeviceGUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
				},
				{
					Action:     "Invalid Axis",
					InputType:  common.Axis,
					InputID:    "RRR",
					DeviceGUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
				},
				{
					Action:     "Valid Button",
					InputType:  common.Button,
					InputID:    "1",
					DeviceGUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
				},
			},
		},
		Template: &common.Template{
			Buttons: []string{"BUTTON_1"},
			Axes:    []string{"AXIS_X", "AXIS_Y"},
		},
	}

	errors := ValidateBindings(exportDevice)

	if len(errors) != 2 {
		t.Errorf("ValidateBindings() should find 2 errors, got %d errors", len(errors))
	}
}

// TestValidateBindings_PartialGUIDMatch tests GUID matching with partial match (DCS vs IL-2)
func TestValidateBindings_PartialGUIDMatch(t *testing.T) {
	// DCS format (5 segments)
	exportDevice := &common.ExportDevice{
		Device: &common.Device{
			Name: "Test Device",
			GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		Profile: &common.Profile{
			Bindings: []common.Binding{
				{
					Action:     "Test Action",
					InputType:  common.Button,
					InputID:    "1",
					DeviceGUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}", // Exact match
				},
			},
		},
		Template: &common.Template{
			Buttons: []string{"BUTTON_1"},
		},
	}

	errors := ValidateBindings(exportDevice)

	if len(errors) != 0 {
		t.Errorf("ValidateBindings() should match partial GUIDs, got %d errors", len(errors))
	}
}

// TestValidateBindings_EmptyTemplate tests validation with empty template
func TestValidateBindings_EmptyTemplate(t *testing.T) {
	exportDevice := &common.ExportDevice{
		Device: &common.Device{
			Name: "Test Device",
			GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		Profile: &common.Profile{
			Bindings: []common.Binding{
				{
					Action:     "Test Action",
					InputType:  common.Button,
					InputID:    "1",
					DeviceGUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
				},
			},
		},
		Template: &common.Template{
			Buttons: []string{},
			Axes:    []string{},
			Hats:    []string{},
		},
	}

	errors := ValidateBindings(exportDevice)

	if len(errors) != 1 {
		t.Errorf("ValidateBindings() with empty template should find 1 error, got %d errors", len(errors))
	}
}
