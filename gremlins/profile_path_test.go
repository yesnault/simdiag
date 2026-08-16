package gremlins

import (
	"os"
	"path/filepath"
	"testing"

	"simdiag/common"
)

const testDeviceGUID = "{EE6F1C30-3F2E-11F0-8001-444553540000}"

// writeTestProfile writes a minimal Gremlins profile binding one button on
// testDeviceGUID, and returns its path.
func writeTestProfile(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "profile.xml")
	content := `<?xml version="1.0"?>
<profile version="1">
    <devices>
        <device device-guid="` + testDeviceGUID + `" name="Test Joystick" type="joystick">
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
</profile>`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return path
}

// TestGetProfilePath covers the shape the GUI's Configuration tab writes: one
// Gremlins path per simulator, applying to every DCS module.
func TestGetProfilePath(t *testing.T) {
	const dcsProfile = `C:\gremlins\all.xml`

	config := &common.Config{Simulators: map[string]*common.SimulatorConfig{
		"dcs_world":     {DCSPath: `C:\DCS`, GremlinsProfileFilepath: dcsProfile},
		"il2_sturmovik": {IL2InputPath: `C:\IL-2`},
	}}

	if got := GetProfilePath(config, common.DCSWorld); got != dcsProfile {
		t.Errorf("GetProfilePath(DCS) = %q, want %q", got, dcsProfile)
	}
	if got := GetProfilePath(config, common.IL2Sturmovik); got != "" {
		t.Errorf("GetProfilePath(IL-2) = %q, want empty", got)
	}
}

// A simulator the user never set up must not be created just by being asked about.
func TestGetProfilePath_UnconfiguredSimulator(t *testing.T) {
	config := &common.Config{}

	if got := GetProfilePath(config, common.IL2Korea); got != "" {
		t.Errorf("GetProfilePath = %q, want empty", got)
	}
	if len(config.Simulators) != 0 {
		t.Errorf("lookup created a simulator section: %v", config.Simulators)
	}
}

// TestLoadBindingsForDevice_DeduplicatesProfiles pins that a profile shared by
// several simulators (the usual setup, since one Gremlins profile covers the
// whole rig) is not parsed once per simulator that names it.
func TestLoadBindingsForDevice_DeduplicatesProfiles(t *testing.T) {
	profile := writeTestProfile(t)

	config := &common.Config{Simulators: map[string]*common.SimulatorConfig{
		"dcs_world":     {DCSPath: `C:\DCS`, GremlinsProfileFilepath: profile},
		"il2_sturmovik": {IL2InputPath: `C:\IL-2`, GremlinsProfileFilepath: profile},
		"il2_korea":     {IL2InputPath: `C:\Korea`, GremlinsProfileFilepath: profile},
	}}

	shared := LoadBindingsForDevice(testDeviceGUID, config)
	if len(shared) == 0 {
		t.Fatal("expected the fixture profile to yield at least one binding")
	}

	// The same profile named by one simulator must produce the same result.
	single := &common.Config{Simulators: map[string]*common.SimulatorConfig{
		"dcs_world": {DCSPath: `C:\DCS`, GremlinsProfileFilepath: profile},
	}}
	if got, want := len(shared), len(LoadBindingsForDevice(testDeviceGUID, single)); got != want {
		t.Errorf("three simulators sharing one profile returned %d bindings, want %d", got, want)
	}
}

func TestLoadBindingsForDevice_NoConfig(t *testing.T) {
	if got := LoadBindingsForDevice(testDeviceGUID, nil); got != nil {
		t.Errorf("LoadBindingsForDevice(nil config) = %v, want nil", got)
	}
	if got := LoadBindingsForDevice(testDeviceGUID, &common.Config{}); got != nil {
		t.Errorf("LoadBindingsForDevice(empty config) = %v, want nil", got)
	}
}
