package gui

import (
	"simdiag/common"
	"slices"
	"testing"
)

// CheckCompatibility had no production caller and no test until the GUI started
// ranking templates with it. These pin the behaviour it is now trusted for.

func compatTemplate() *common.Template {
	return &common.Template{
		Name:    "test.svg",
		Buttons: []string{"BUTTON_1", "BUTTON_2", "BUTTON_25"},
		Axes:    []string{"AXIS_X", "AXIS_Y"},
		Hats:    []string{"POV_1_U"},
	}
}

func compatProfile(bindings ...common.Binding) *common.Profile {
	return &common.Profile{Bindings: bindings}
}

func binding(guid string, inputType common.InputType, inputID string) common.Binding {
	return common.Binding{DeviceGUID: guid, InputType: inputType, InputID: inputID, Action: "action"}
}

func TestCheckCompatibility_ScoresMatchingInputs(t *testing.T) {
	device := &common.Device{GUID: "guid-a", Name: "Test"}
	profile := compatProfile(
		binding("guid-a", common.Button, "1"),
		binding("guid-a", common.Button, "2"),
		binding("guid-a", common.Axis, "x"),
	)

	compatible, score, missing := CheckCompatibility(device, profile, compatTemplate())

	if score != 3 {
		t.Errorf("score = %d, want 3", score)
	}
	if len(missing) != 0 {
		t.Errorf("missing = %v, want none", missing)
	}
	// 3 of 6 template keys is exactly the 0.5 threshold.
	if !compatible {
		t.Error("compatible = false, want true at the 50% threshold")
	}
}

// A binding on another device must not contribute to this device's score.
func TestCheckCompatibility_IgnoresOtherDevices(t *testing.T) {
	device := &common.Device{GUID: "guid-a", Name: "Test"}
	profile := compatProfile(
		binding("guid-a", common.Button, "1"),
		binding("guid-b", common.Button, "2"),
		binding("guid-b", common.Axis, "x"),
	)

	_, score, _ := CheckCompatibility(device, profile, compatTemplate())

	if score != 1 {
		t.Errorf("score = %d, want 1 (only the guid-a binding counts)", score)
	}
}

func TestCheckCompatibility_ReportsInputsAbsentFromTemplate(t *testing.T) {
	device := &common.Device{GUID: "guid-a", Name: "Test"}
	profile := compatProfile(
		binding("guid-a", common.Button, "1"),
		binding("guid-a", common.Button, "99"), // not in the template
	)

	_, _, missing := CheckCompatibility(device, profile, compatTemplate())

	if !slices.Contains(missing, "BUTTON_99") {
		t.Errorf("missing = %v, want it to contain BUTTON_99", missing)
	}
	if slices.Contains(missing, "BUTTON_1") {
		t.Errorf("missing = %v, must not contain a key the template has", missing)
	}
}

// A switch-off binding (BTN25_OFF) targets the same template key as BTN25.
// Reporting it as missing would flag healthy configurations as broken.
func TestCheckCompatibility_StripsOffSuffix(t *testing.T) {
	device := &common.Device{GUID: "guid-a", Name: "Test"}
	profile := compatProfile(binding("guid-a", common.Button, "25_OFF"))

	_, score, missing := CheckCompatibility(device, profile, compatTemplate())

	if score != 1 {
		t.Errorf("score = %d, want 1: BUTTON_25_OFF maps to the BUTTON_25 key", score)
	}
	if len(missing) != 0 {
		t.Errorf("missing = %v, want none", missing)
	}
}

func TestCheckCompatibility_IsCaseInsensitiveOnAxesAndHats(t *testing.T) {
	device := &common.Device{GUID: "guid-a", Name: "Test"}
	profile := compatProfile(
		binding("guid-a", common.Axis, "x"),
		binding("guid-a", common.Hat, "1_u"),
	)

	_, score, missing := CheckCompatibility(device, profile, compatTemplate())

	if score != 2 {
		t.Errorf("score = %d, want 2 (lowercase axis/hat ids must still match)", score)
	}
	if len(missing) != 0 {
		t.Errorf("missing = %v, want none", missing)
	}
}

func TestCheckCompatibility_IncompatibleBelowHalf(t *testing.T) {
	device := &common.Device{GUID: "guid-a", Name: "Test"}
	profile := compatProfile(binding("guid-a", common.Button, "1"))

	compatible, score, _ := CheckCompatibility(device, profile, compatTemplate())

	if score != 1 {
		t.Fatalf("score = %d, want 1", score)
	}
	if compatible {
		t.Error("compatible = true, want false: 1 of 6 keys is below the threshold")
	}
}

// An empty template cannot contradict anything, so it counts as compatible.
func TestCheckCompatibility_EmptyTemplate(t *testing.T) {
	device := &common.Device{GUID: "guid-a", Name: "Test"}
	profile := compatProfile(binding("guid-a", common.Button, "1"))

	compatible, score, _ := CheckCompatibility(device, profile, &common.Template{Name: "empty.svg"})

	if score != 0 {
		t.Errorf("score = %d, want 0", score)
	}
	if !compatible {
		t.Error("compatible = false, want true for a template with no keys")
	}
}
