package srs

import (
	"os"
	"path/filepath"
	"testing"

	"simdiag/common"
)

// One SimpleRadio installation serves both IL-2 titles, and a different one
// serves DCS: that is the whole shape of the setting.
func TestSRSPathFor(t *testing.T) {
	config := &common.Config{
		DCSSRSPath: `C:\Program Files\DCS-SimpleRadio-Standalone`,
		IL2SRSPath: `C:\Program Files\IL2-SimpleRadio-Standalone`,
	}

	if got := config.SRSPathFor(common.DCSWorld); got != config.DCSSRSPath {
		t.Errorf("SRSPathFor(DCS) = %q, want %q", got, config.DCSSRSPath)
	}

	korea := config.SRSPathFor(common.IL2Korea)
	greatBattles := config.SRSPathFor(common.IL2Sturmovik)
	if korea != config.IL2SRSPath || greatBattles != config.IL2SRSPath {
		t.Errorf("the two IL-2 titles got %q and %q, want both on %q", greatBattles, korea, config.IL2SRSPath)
	}
}

// il2SRSFixture writes an IL2-SRS installation holding one binding for guid.
func il2SRSFixture(t *testing.T, guid string) string {
	t.Helper()

	dir := t.TempDir()
	content := "[Switch1]\nname=\"Radio Switch\"\nbutton=17\nguid=" + guid + "\n"
	if err := os.WriteFile(filepath.Join(dir, "default.cfg"), []byte(content), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return dir
}

// SRS records 5-segment GUIDs while IL-2 uses 4-segment ones, so matching them
// by equality dropped every IL-2 radio binding on the floor.
func TestEnrich_MatchesAnIL2DeviceAgainstASRSGUID(t *testing.T) {
	const (
		il2GUID = "530648c0-98c7-11f0-0000545345440280"
		srsGUID = "530648c0-98c7-11f0-8002-444553540000"
	)

	config := &common.Config{IL2SRSPath: il2SRSFixture(t, srsGUID)}

	exportDevice := &common.ExportDevice{
		Device: &common.Device{GUID: il2GUID, Name: "WINWING Orion Throttle"},
		Profile: &common.Profile{
			Devices: map[string]*common.Device{il2GUID: {GUID: il2GUID, Name: "WINWING Orion Throttle"}},
		},
	}

	NewEnricher().Enrich(exportDevice, &common.Profile{SimType: common.IL2Sturmovik}, config)

	if len(exportDevice.Profile.Bindings) != 1 {
		t.Fatalf("got %d bindings, want the SRS one", len(exportDevice.Profile.Bindings))
	}

	binding := exportDevice.Profile.Bindings[0]
	if binding.Action != "SRS: Switch1" || binding.InputID != "18" {
		t.Errorf("binding = %+v, want SRS: Switch1 on button 18 (SRS counts from 0)", binding)
	}
	// The binding has to carry the simulator's own device identity: it is what
	// the CSV writes as the physical device.
	if binding.DeviceGUID != il2GUID {
		t.Errorf("DeviceGUID = %q, want the simulator's %q rather than the one SRS saw", binding.DeviceGUID, il2GUID)
	}
}

func TestEnrich_DoesNothingWithoutASRSPath(t *testing.T) {
	exportDevice := &common.ExportDevice{
		Device:  &common.Device{GUID: "530648c0-98c7-11f0-0000545345440280"},
		Profile: &common.Profile{Devices: map[string]*common.Device{}},
	}

	NewEnricher().Enrich(exportDevice, &common.Profile{SimType: common.IL2Korea}, &common.Config{})

	if len(exportDevice.Profile.Bindings) != 0 {
		t.Errorf("got %d bindings, want none when SRS is not configured", len(exportDevice.Profile.Bindings))
	}
}

// A different device must not pick up another one's radio bindings.
func TestEnrich_LeavesUnrelatedDevicesAlone(t *testing.T) {
	config := &common.Config{IL2SRSPath: il2SRSFixture(t, "530648c0-98c7-11f0-8002-444553540000")}

	const otherGUID = "ee6f1c30-3f2e-11f0-0000545345440180"
	exportDevice := &common.ExportDevice{
		Device: &common.Device{GUID: otherGUID, Name: "Arduino Due"},
		Profile: &common.Profile{
			Devices: map[string]*common.Device{otherGUID: {GUID: otherGUID, Name: "Arduino Due"}},
		},
	}

	NewEnricher().Enrich(exportDevice, &common.Profile{SimType: common.IL2Sturmovik}, config)

	if len(exportDevice.Profile.Bindings) != 0 {
		t.Errorf("got %d bindings, want none: this is another device", len(exportDevice.Profile.Bindings))
	}
}
