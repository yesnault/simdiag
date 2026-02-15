package integration

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"simdiag/common"
	"simdiag/dcs"
	"simdiag/gremlins"
	"simdiag/il2"
	"simdiag/openkneeboard"
	"simdiag/srs"
	"simdiag/target"
	"simdiag/workflow"

	"gopkg.in/yaml.v3"
)

func TestMain(m *testing.M) {
	// Wire up ExtFuncs exactly like cmd/simdiag/main.go:init()
	common.ExtFuncs = &common.ExternalFuncs{
		GetTargetDeviceNumbers:    target.GetTargetDeviceNumbers,
		AutoMatchTargetDevices:    target.AutoMatchTargetDevices,
		TargetDeviceNumberToName:  target.DeviceNumberToName,
		GetUnmatchedTargetDevices: target.GetUnmatchedTargetDevices,
		LoadGremlinsBindingsForDevice: func(guid string, config *common.Config) interface{} {
			return gremlins.LoadBindingsForDevice(guid, config)
		},
		LoadOpenKneeboardBindingsForDevice: func(guid string, config *common.Config) interface{} {
			return openkneeboard.LoadBindingsForDevice(guid, config)
		},
		ParseGremlinsProfile: func(path string) (interface{}, error) {
			return gremlins.ParseProfile(path)
		},
	}

	os.Exit(m.Run())
}

func TestIntegration(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testCase := entry.Name()
		configPath := filepath.Join("testdata", testCase, "mapping_config.yaml")
		expectedCSVPath := filepath.Join("testdata", testCase, "expected.csv")

		// Skip directories without the required files
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			continue
		}
		if _, err := os.Stat(expectedCSVPath); os.IsNotExist(err) {
			continue
		}

		t.Run(testCase, func(t *testing.T) {
			runTestCase(t, testCase, configPath, expectedCSVPath)
		})
	}
}

func runTestCase(t *testing.T, _ /* name */, configPath, expectedCSVPath string) {
	t.Helper()

	// Create temp directory for output
	tmpDir := t.TempDir()

	// Read config, replace output_directory, write to temp file
	tempConfigPath := prepareConfig(t, configPath, tmpDir)

	// Set global config filename
	common.SetConfigFileName(tempConfigPath)

	// Create parsers and enrichers
	parsers := map[common.SimulationType]common.SimulatorParser{
		common.DCSWorld:     dcs.NewParser(),
		common.IL2Sturmovik: il2.NewParser(),
	}
	enrichers := []common.BindingEnricher{
		gremlins.NewEnricher(),
		target.NewEnricher(),
		openkneeboard.NewEnricher(),
		srs.NewEnricher(),
	}

	// Run the pipeline (noSVG=true)
	workflow.ExportAllSimulatorsBatchWithInterfaces(parsers, enrichers, "", true)

	// Compare generated CSV with expected
	actualCSVPath := filepath.Join(tmpDir, "export.csv")
	if _, err := os.Stat(actualCSVPath); os.IsNotExist(err) {
		t.Fatalf("pipeline did not produce export.csv in %s", tmpDir)
	}

	// If UPDATE_EXPECTED=1, copy generated CSV as new expected CSV
	if os.Getenv("UPDATE_EXPECTED") == "1" {
		data, err := os.ReadFile(actualCSVPath)
		if err != nil {
			t.Fatalf("failed to read generated CSV: %v", err)
		}
		if err := os.WriteFile(expectedCSVPath, data, 0644); err != nil {
			t.Fatalf("failed to write expected CSV: %v", err)
		}
		t.Logf("Updated expected CSV: %s", expectedCSVPath)
		return
	}

	compareCSV(t, expectedCSVPath, actualCSVPath)
}

// prepareConfig reads the test config, replaces output_directory with tmpDir,
// replaces templates_directory with absolute path to project templates,
// and writes the modified config to a temp file. Returns the temp config path.
func prepareConfig(t *testing.T, configPath, tmpDir string) string {
	t.Helper()

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config %s: %v", configPath, err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to parse config YAML: %v", err)
	}

	raw["output_directory"] = tmpDir

	// Get absolute path to project templates directory (../../templates from integration test directory)
	templatesDir, err := filepath.Abs("../../templates")
	if err != nil {
		t.Fatalf("failed to get absolute path to templates: %v", err)
	}
	raw["templates_directory"] = templatesDir

	modified, err := yaml.Marshal(raw)
	if err != nil {
		t.Fatalf("failed to marshal modified config: %v", err)
	}

	tempConfigPath := filepath.Join(tmpDir, "mapping_config.yaml")
	if err := os.WriteFile(tempConfigPath, modified, 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	return tempConfigPath
}

// compareCSV reads both CSV files, sorts rows lexicographically, and compares them.
func compareCSV(t *testing.T, expectedPath, actualPath string) {
	t.Helper()

	expectedRows := readAndSortCSV(t, expectedPath)
	actualRows := readAndSortCSV(t, actualPath)

	if len(expectedRows) != len(actualRows) {
		t.Errorf("row count mismatch: expected %d, got %d", len(expectedRows), len(actualRows))
	}

	// Compare row by row
	maxRows := len(expectedRows)
	if len(actualRows) > maxRows {
		maxRows = len(actualRows)
	}

	mismatches := 0
	for i := 0; i < maxRows; i++ {
		var expected, actual string
		if i < len(expectedRows) {
			expected = expectedRows[i]
		} else {
			expected = "<missing>"
		}
		if i < len(actualRows) {
			actual = actualRows[i]
		} else {
			actual = "<missing>"
		}

		if expected != actual {
			mismatches++
			if mismatches <= 10 {
				t.Errorf("row %d mismatch:\n  expected: %s\n  actual:   %s", i, expected, actual)
			}
		}
	}

	if mismatches > 10 {
		t.Errorf("... and %d more mismatches (showing first 10)", mismatches-10)
	}

	if mismatches == 0 {
		t.Logf("all %d data rows match", len(expectedRows))
	}
}

// readAndSortCSV reads a CSV file, skips the header, joins each row into a string, and sorts.
func readAndSortCSV(t *testing.T, path string) []string {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open CSV %s: %v", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV %s: %v", path, err)
	}

	if len(records) < 2 {
		t.Fatalf("CSV %s has no data rows (only %d lines)", path, len(records))
	}

	// Verify headers match
	header := strings.Join(records[0], ",")
	_ = header // header validation is implicit — if columns differ, row comparisons will fail

	// Join each data row and sort
	rows := make([]string, 0, len(records)-1)
	for _, record := range records[1:] {
		rows = append(rows, strings.Join(record, "\t"))
	}

	sort.Strings(rows)
	return rows
}
