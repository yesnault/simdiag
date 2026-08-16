package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"simdiag/common"
)

func fullConfig() *common.Config {
	return &common.Config{
		TemplatesDirectory:            "templates",
		OutputDirectory:               "output",
		DrawIOPath:                    `C:\Program Files\draw.io\draw.io.exe`,
		OpenKneeboardProfilesFilepath: `C:\okb\Profiles.json`,
		DCSSRSPath:                    `C:\Program Files\DCS-SimpleRadio-Standalone`,
		IL2SRSPath:                    `C:\Program Files\IL2-SimpleRadio-Standalone`,
		DeviceMappings: []common.DeviceTemplateMapping{
			{DeviceGUID: testGUID, DeviceName: "Arduino Due", TemplateFilepath: "due.svg"},
		},
		Simulators: map[string]*common.SimulatorConfig{
			"dcs_world": {
				DCSPath:                 `C:\Users\me\Saved Games\DCS`,
				GremlinsProfileFilepath: `C:\gremlins.xml`,
				TargetProfileFilepath:   `C:\target.tmc`,
			},
			"il2_sturmovik": {
				IL2InputPath: `C:\il2\data\input`,
			},
		},
	}
}

const testGUID = "{EE6F1C30-3F2E-11F0-8001-444553540000}"

func TestConfigDTO_RoundTripPreservesEveryField(t *testing.T) {
	original := fullConfig()

	// Project to the form shape and straight back, as a save with no edits does.
	applied := fullConfig()
	applyDTO(applied, toDTO(original))

	if applied.TemplatesDirectory != original.TemplatesDirectory ||
		applied.OutputDirectory != original.OutputDirectory ||
		applied.DrawIOPath != original.DrawIOPath ||
		applied.OpenKneeboardProfilesFilepath != original.OpenKneeboardProfilesFilepath ||
		applied.DCSSRSPath != original.DCSSRSPath ||
		applied.IL2SRSPath != original.IL2SRSPath {
		t.Errorf("round trip lost a global field: %+v", applied)
	}

	dcs := applied.Simulators["dcs_world"]
	if dcs.DCSPath != original.Simulators["dcs_world"].DCSPath {
		t.Errorf("DCSPath = %q, want %q", dcs.DCSPath, original.Simulators["dcs_world"].DCSPath)
	}
	if dcs.GremlinsProfileFilepath == "" || dcs.TargetProfileFilepath == "" {
		t.Errorf("round trip lost a DCS tool path: %+v", dcs)
	}
	if il2 := applied.Simulators["il2_sturmovik"]; il2.IL2InputPath != `C:\il2\data\input` {
		t.Errorf("IL2InputPath = %q", il2.IL2InputPath)
	}
}

// The Configuration form does not show device mappings, so saving it must leave
// them exactly as they were.
func TestApplyDTO_PreservesFieldsOutsideTheForm(t *testing.T) {
	config := fullConfig()

	applyDTO(config, toDTO(config))

	if len(config.DeviceMappings) != 1 || config.DeviceMappings[0].DeviceName != "Arduino Due" {
		t.Errorf("device mappings were altered: %+v", config.DeviceMappings)
	}
}

// Saving a form where a simulator was left blank must not create an empty
// section, otherwise the YAML fills up with noise the user never asked for.
func TestApplyDTO_DoesNotCreateEmptySimulatorSections(t *testing.T) {
	config := &common.Config{Simulators: map[string]*common.SimulatorConfig{}}

	applyDTO(config, configDTO{
		TemplatesDirectory: "templates",
		Simulators: []simulatorDTO{
			{Key: "dcs_world", Path: `C:\dcs`},
			{Key: "il2_sturmovik"},
			{Key: "il2_korea"},
		},
	})

	if _, exists := config.Simulators["il2_sturmovik"]; exists {
		t.Error("an empty simulator section was created for il2_sturmovik")
	}
	if _, exists := config.Simulators["dcs_world"]; !exists {
		t.Error("the configured DCS section is missing")
	}
}

// Clearing a field in the form must actually clear it in the config.
func TestApplyDTO_ClearsRemovedValues(t *testing.T) {
	config := fullConfig()

	dto := toDTO(config)
	dto.DrawIOPath = ""
	dto.IL2SRSPath = ""
	for i := range dto.Simulators {
		if dto.Simulators[i].Key == "dcs_world" {
			dto.Simulators[i].GremlinsProfileFilepath = ""
		}
	}
	applyDTO(config, dto)

	if config.DrawIOPath != "" {
		t.Errorf("DrawIOPath = %q, want it cleared", config.DrawIOPath)
	}
	if config.IL2SRSPath != "" {
		t.Errorf("IL2SRSPath = %q, want it cleared", config.IL2SRSPath)
	}
	if got := config.Simulators["dcs_world"].GremlinsProfileFilepath; got != "" {
		t.Errorf("DCS GremlinsProfileFilepath = %q, want it cleared", got)
	}
}

// Reading a simulator's section must not create it. The GUI kept a private copy
// of this lookup precisely because Config's own getter inserted an empty entry,
// which a later save then wrote into the user's YAML. The getter is now named
// EnsureSimulatorConfig and the copy is gone.
func TestLookupSimulatorConfig_DoesNotCreateEntries(t *testing.T) {
	config := &common.Config{Simulators: map[string]*common.SimulatorConfig{}}

	if section := config.LookupSimulatorConfig(common.IL2Korea); section != nil {
		t.Errorf("LookupSimulatorConfig returned %+v, want nil", section)
	}
	if len(config.Simulators) != 0 {
		t.Errorf("LookupSimulatorConfig created %d entries as a side effect", len(config.Simulators))
	}
}

func TestComputeStatus_FlagsMissingRequiredPaths(t *testing.T) {
	status := computeStatus(&common.Config{})

	if status.Templates.Severity != "error" {
		t.Errorf("templates severity = %q, want error when unset", status.Templates.Severity)
	}
	if status.Output.Severity != "error" {
		t.Errorf("output severity = %q, want error when unset", status.Output.Severity)
	}
}

func TestComputeStatus_CountsTemplates(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.svg")
	writeFile(t, dir, "b.SVG") // extension match must be case-insensitive
	writeFile(t, dir, "notes.txt")

	status := computeStatus(&common.Config{TemplatesDirectory: dir})

	if status.Templates.Severity != "" || !status.Templates.Exists {
		t.Fatalf("templates status = %+v, want a clean result", status.Templates)
	}
	// The detail is a message code plus its values: the sentence itself, in both
	// languages, lives in the frontend catalogue.
	if status.Templates.Detail.Code != msgTemplatesFound {
		t.Errorf("templates detail = %+v, want %s", status.Templates.Detail, msgTemplatesFound)
	}
	if got := status.Templates.Detail.Args["count"]; got != "2" {
		t.Errorf("templates count = %q, want 2", got)
	}
}

// An output directory that does not exist yet is normal: the export creates it.
func TestComputeStatus_MissingOutputDirectoryIsNotAnError(t *testing.T) {
	status := computeStatus(&common.Config{OutputDirectory: "does-not-exist-yet"})

	if status.Output.Severity != "" {
		t.Errorf("output severity = %q, want no severity for a directory to be created", status.Output.Severity)
	}
}

func TestConfigRoute_SaveWritesToDisk(t *testing.T) {
	state := newTestState(t, &common.Config{})
	h := NewHandler(state)

	body, err := json.Marshal(configDTO{
		TemplatesDirectory: "templates",
		OutputDirectory:    "output",
		DrawIOPath:         `C:\draw.io.exe`,
		Simulators:         []simulatorDTO{{Key: "dcs_world", Path: `C:\dcs`}},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(string(body))))

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/config = %d (%s), want 200", rec.Code, rec.Body.String())
	}

	// Re-read from disk: the response must reflect what was persisted.
	written, err := common.LoadConfigFrom(state.ConfigPath())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if written.TemplatesDirectory != "templates" || written.DrawIOPath != `C:\draw.io.exe` {
		t.Errorf("saved config = %+v", written)
	}
	if written.Simulators["dcs_world"] == nil || written.Simulators["dcs_world"].DCSPath != `C:\dcs` {
		t.Errorf("saved DCS section = %+v", written.Simulators["dcs_world"])
	}
}

func TestConfigRoute_RejectsInvalidPayload(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader("not json")))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("PUT /api/config with junk = %d, want 400", rec.Code)
	}
}
