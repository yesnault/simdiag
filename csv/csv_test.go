package csv

import (
	"os"
	"path/filepath"
	"simdiag/common"
	"strings"
	"testing"
)

// TestGetHeaderString tests CSV header generation
func TestGetHeaderString(t *testing.T) {
	header := GetHeaderString()

	// Check that header ends with newline
	if !strings.HasSuffix(header, "\n") {
		t.Error("GetHeaderString() should end with newline")
	}

	// Check that all columns are present
	expectedColumns := []string{
		"Simulator",
		"Module",
		"Action",
		"Modifier",
		"Modifier Device",
		"Modifier Num",
		"Physical Device",
		"Physical Input",
		"Physical Device GUID",
		"Virtual Device",
		"Virtual Input",
		"Template Key",
		"Template",
	}

	headerWithoutNewline := strings.TrimSuffix(header, "\n")
	columns := strings.Split(headerWithoutNewline, ",")

	if len(columns) != len(expectedColumns) {
		t.Errorf("GetHeaderString() returned %d columns, want %d", len(columns), len(expectedColumns))
	}

	for i, expected := range expectedColumns {
		if i >= len(columns) {
			t.Errorf("Missing column at index %d: %q", i, expected)
			continue
		}
		if columns[i] != expected {
			t.Errorf("Column %d = %q, want %q", i, columns[i], expected)
		}
	}
}

// TestAllColumns tests that AllColumns contains all expected columns
func TestAllColumns(t *testing.T) {
	expected := 13 // Number of columns defined
	if len(AllColumns) != expected {
		t.Errorf("AllColumns has %d elements, want %d", len(AllColumns), expected)
	}

	// Check for duplicate columns
	seen := make(map[string]bool)
	for _, col := range AllColumns {
		if seen[col] {
			t.Errorf("Duplicate column in AllColumns: %q", col)
		}
		seen[col] = true
	}
}

// TestExportToCSV_EmptyDevices tests CSV export with no devices
func TestExportToCSV_EmptyDevices(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "export.csv")

	err := ExportToCSV([]*common.ExportDevice{}, outputPath, nil)
	if err != nil {
		t.Errorf("ExportToCSV() with empty devices should not error, got: %v", err)
	}

	// File should not be created for empty devices
	if _, err := os.Stat(outputPath); err == nil {
		t.Error("ExportToCSV() should not create file for empty devices")
	}
}

// TestExportToCSV_BasicBinding tests CSV export with a simple binding
func TestExportToCSV_BasicBinding(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "export.csv")

	// Create a minimal export device with one binding
	exportDevices := []*common.ExportDevice{
		{
			Device: &common.Device{
				Name: "Test Joystick",
				GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			},
			Profile: &common.Profile{
				Name:    "Test Profile",
				SimType: common.DCSWorld,
				Module:  "m2000c",
				Bindings: []common.Binding{
					{
						Action:     "Test Action",
						InputType:  common.Button,
						InputID:    "1",
						DeviceName: "Test Joystick",
					},
				},
			},
		},
	}

	err := ExportToCSV(exportDevices, outputPath, nil)
	if err != nil {
		t.Fatalf("ExportToCSV() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("ExportToCSV() did not create output file: %v", err)
	}

	// Read and verify file content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)

	// Check header is present
	if !strings.Contains(contentStr, "Simulator,Module,Action") {
		t.Error("Output file missing CSV header")
	}

	// Check that binding data is present
	if !strings.Contains(contentStr, "dcs_world") {
		t.Error("Output file missing simulator (dcs_world)")
	}
	if !strings.Contains(contentStr, "m2000c") {
		t.Error("Output file missing module (m2000c)")
	}
	if !strings.Contains(contentStr, "Test Action") {
		t.Error("Output file missing action (Test Action)")
	}
	if !strings.Contains(contentStr, "Button_1") {
		t.Error("Output file missing template key (Button_1)")
	}
}

// TestExportToCSV_SkipKeyboardBindings tests that keyboard bindings are skipped
func TestExportToCSV_SkipKeyboardBindings(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "export.csv")

	exportDevices := []*common.ExportDevice{
		{
			Device: &common.Device{
				Name: "Keyboard",
				GUID: "{00000000-0000-0000-0000-000000000000}",
			},
			Profile: &common.Profile{
				Name:    "Test Profile",
				SimType: common.DCSWorld,
				Module:  "m2000c",
				Bindings: []common.Binding{
					{
						Action:     "Keyboard Action",
						InputType:  common.Button,
						InputID:    "1",
						DeviceName: "Keyboard",
					},
				},
			},
		},
	}

	err := ExportToCSV(exportDevices, outputPath, nil)
	if err != nil {
		t.Fatalf("ExportToCSV() error = %v", err)
	}

	// Read file content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)

	// Check that keyboard action is NOT present (skipped)
	if strings.Contains(contentStr, "Keyboard Action") {
		t.Error("Keyboard bindings should be skipped, but found in output")
	}
}

// TestExportToCSV_MultipleInputTypes tests CSV export with different input types
func TestExportToCSV_MultipleInputTypes(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "export.csv")

	exportDevices := []*common.ExportDevice{
		{
			Device: &common.Device{
				Name: "Test Joystick",
				GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			},
			Profile: &common.Profile{
				Name:    "Test Profile",
				SimType: common.DCSWorld,
				Module:  "m2000c",
				Bindings: []common.Binding{
					{
						Action:     "Button Action",
						InputType:  common.Button,
						InputID:    "1",
						DeviceName: "Test Joystick",
					},
					{
						Action:     "Axis Action",
						InputType:  common.Axis,
						InputID:    "X",
						DeviceName: "Test Joystick",
					},
					{
						Action:     "Hat Action",
						InputType:  common.Hat,
						InputID:    "1_U",
						DeviceName: "Test Joystick",
					},
				},
			},
		},
	}

	err := ExportToCSV(exportDevices, outputPath, nil)
	if err != nil {
		t.Fatalf("ExportToCSV() error = %v", err)
	}

	// Read file content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)

	// Check template keys for different input types
	tests := []struct {
		name        string
		templateKey string
	}{
		{"button", "Button_1"},
		{"axis", "AXIS_X"},
		{"hat", "POV_1_U"},
	}

	for _, tt := range tests {
		if !strings.Contains(contentStr, tt.templateKey) {
			t.Errorf("Missing template key for %s: %s", tt.name, tt.templateKey)
		}
	}
}

// TestExportToCSV_IL2Simulator tests CSV export for IL-2
func TestExportToCSV_IL2Simulator(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "export.csv")

	exportDevices := []*common.ExportDevice{
		{
			Device: &common.Device{
				Name: "Test Joystick",
				GUID: "{EE6F1C30-3F2E-11F0-0000545345440180}",
			},
			Profile: &common.Profile{
				Name:    "Test Profile",
				SimType: common.IL2Sturmovik,
				Bindings: []common.Binding{
					{
						Action:     "IL-2 Action",
						InputType:  common.Button,
						InputID:    "1",
						DeviceName: "Test Joystick",
					},
				},
			},
		},
	}

	err := ExportToCSV(exportDevices, outputPath, nil)
	if err != nil {
		t.Fatalf("ExportToCSV() error = %v", err)
	}

	// Read file content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)

	// Check IL-2 specific values
	if !strings.Contains(contentStr, "il2_sturmovik") {
		t.Error("Output file missing simulator (il2_sturmovik)")
	}
	if !strings.Contains(contentStr, "il2") {
		t.Error("Output file missing module (il2)")
	}
}
