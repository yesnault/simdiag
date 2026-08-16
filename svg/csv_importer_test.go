package svg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"simdiag/common"
)

const testStickGUID = "{EE6F1C30-3F2E-11F0-8001-444553540000}"
const testRudderGUID = "{B0C891C0-3F30-11F0-8003-444553540000}"

// writeCSV writes an export with one row per (device, template) pair given.
func writeCSV(t *testing.T, rows [][2]string) string {
	t.Helper()

	var b strings.Builder
	b.WriteString("Simulator,Module,Action,Modifier,Modifier Device,Modifier Num," +
		"Physical Device,Physical Input,Physical Device GUID,Virtual Device,Virtual Input," +
		"Template Key,Template\n")
	for i, row := range rows {
		name, guid := row[0], row[1]
		// Only the device columns and the (deliberately empty) template matter here.
		b.WriteString("DCS World,M-2000C,Action" + string(rune('A'+i)) + ",,,," +
			name + ",BTN1," + guid + ",,,Button_1,\n")
	}

	path := filepath.Join(t.TempDir(), "export.csv")
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return path
}

// A device the user marked "skip" has no template on purpose, so reporting it as
// missing on every export contradicts the choice they made.
func TestReadCSVGroups_SkippedDeviceIsNotReportedAsUntemplated(t *testing.T) {
	csvPath := writeCSV(t, [][2]string{
		{"T-Rudder", testRudderGUID},
		{"Arduino Due", testStickGUID},
	})

	config := &common.Config{DeviceMappings: []common.DeviceTemplateMapping{
		{DeviceGUID: testRudderGUID, DeviceName: "T-Rudder", SkipTemplate: true},
		{DeviceGUID: testStickGUID, DeviceName: "Arduino Due"},
	}}

	var logged strings.Builder
	common.SetOutput(&logged)
	defer common.SetOutput(nil)

	if _, _, err := readCSVGroups(csvPath, config); err != nil {
		t.Fatalf("readCSVGroups: %v", err)
	}

	out := logged.String()
	if strings.Contains(out, "T-Rudder") {
		t.Errorf("a skipped device was reported as missing a template:\n%s", out)
	}
	// The device with no mapping at all is still worth a word.
	if !strings.Contains(out, "Arduino Due") {
		t.Errorf("an unmapped device should still be reported:\n%s", out)
	}
}
