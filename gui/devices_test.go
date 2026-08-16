package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"simdiag/common"
	"simdiag/target"
)

// The same physical controller is named with a 5-segment GUID by DCS and a
// 4-segment one by IL-2. Both must land on a single row, or the user is asked to
// map the same stick twice.
func TestFindScannedDevice_MatchesAcrossGUIDFormats(t *testing.T) {
	entries := []*scannedDevice{{
		device: &common.Device{GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}", Name: "Arduino Due"},
		guids:  []string{"{EE6F1C30-3F2E-11F0-8001-444553540000}"},
	}}

	if got := findScannedDevice(entries, "EE6F1C30-3F2E-11f0-444553540000"); got == nil {
		t.Error("the IL-2 form of the same GUID did not match the DCS entry")
	}
	if got := findScannedDevice(entries, "{11111111-2222-3333-8001-444553540000}"); got != nil {
		t.Error("an unrelated GUID matched an existing entry")
	}
}

func TestFindScannedDevice_EmptyList(t *testing.T) {
	if got := findScannedDevice(nil, "{EE6F1C30-3F2E-11F0-8001-444553540000}"); got != nil {
		t.Errorf("findScannedDevice(nil, ...) = %v, want nil", got)
	}
}

// The Devices tab reads devices.length before anything else. An empty scan
// must encode as [] rather than null.
func TestScanDevices_DevicesIsNeverNull(t *testing.T) {
	state, _ := deviceTestState(t)

	if devices := scanDevices(state.Config()).Devices; devices == nil {
		t.Error("Devices is nil, which marshals to null and breaks the tab")
	}
}

// buildDeviceEntry puts the best candidate first.
func TestBuildDeviceEntry_RanksTemplatesByFit(t *testing.T) {
	const guid = "guid-a"
	device := &common.Device{GUID: guid, Name: "Test Stick"}

	merged := &common.Profile{Bindings: []common.Binding{
		{DeviceGUID: guid, InputType: common.Button, InputID: "1", Action: "a"},
		{DeviceGUID: guid, InputType: common.Button, InputID: "2", Action: "b"},
		{DeviceGUID: guid, InputType: common.Axis, InputID: "x", Action: "c"},
	}}

	poor := &common.Template{Name: "poor.svg", FilePath: "poor.svg", Buttons: []string{"BUTTON_9"}}
	good := &common.Template{
		Name: "good.svg", FilePath: "good.svg",
		Buttons: []string{"BUTTON_1", "BUTTON_2"},
		Axes:    []string{"AXIS_X"},
	}

	entry := buildDeviceEntry(&common.Config{}, device, []string{guid}, []string{"DCS World"}, merged,
		[]*common.Template{poor, good})

	if len(entry.Templates) != 2 {
		t.Fatalf("got %d template options, want 2", len(entry.Templates))
	}
	if entry.Templates[0].Name != "good.svg" {
		t.Errorf("best template = %q, want good.svg", entry.Templates[0].Name)
	}
	if entry.Templates[0].Score != 3 || !entry.Templates[0].Compatible {
		t.Errorf("best option = %+v, want score 3 and compatible", entry.Templates[0])
	}
	if entry.Bindings != 3 {
		t.Errorf("bindings = %d, want 3", entry.Bindings)
	}
}

// Real case from the Arduino Due: a 42-key button-box template, a 56-key
// joystick template and a 109-key throttle template all covered the same 40
// inputs. Ranking on raw score put the throttle above the template actually made
// for the device, so the tie-break is the share of the template's keys used.
func TestBuildDeviceEntry_PrefersTheTightestTemplateOnEqualScore(t *testing.T) {
	const guid = "guid-a"
	device := &common.Device{GUID: guid, Name: "Button Box"}

	var bindings []common.Binding
	for i := 1; i <= 40; i++ {
		bindings = append(bindings, common.Binding{
			DeviceGUID: guid, InputType: common.Button,
			InputID: strconv.Itoa(i), Action: "a",
		})
	}
	merged := &common.Profile{Bindings: bindings}

	// Both templates contain BUTTON_1..40; the big one adds 60 unused keys.
	buttons := func(n int) []string {
		var keys []string
		for i := 1; i <= n; i++ {
			keys = append(keys, "BUTTON_"+strconv.Itoa(i))
		}
		return keys
	}
	tight := &common.Template{Name: "made-for-it.svg", FilePath: "made-for-it.svg", Buttons: buttons(42)}
	big := &common.Template{Name: "huge-throttle.svg", FilePath: "huge-throttle.svg", Buttons: buttons(100)}

	entry := buildDeviceEntry(&common.Config{}, device, []string{guid}, nil, merged,
		[]*common.Template{big, tight})

	if entry.Templates[0].Score != entry.Templates[1].Score {
		t.Fatalf("test setup broken: scores differ (%d vs %d)",
			entry.Templates[0].Score, entry.Templates[1].Score)
	}
	if entry.Templates[0].Name != "made-for-it.svg" {
		t.Errorf("first suggestion = %q, want made-for-it.svg (42 keys, not 100)",
			entry.Templates[0].Name)
	}
}

func TestBuildDeviceEntry_ReportsExistingMapping(t *testing.T) {
	const guid = "{EE6F1C30-3F2E-11F0-8001-444553540000}"
	config := &common.Config{
		TemplatesDirectory: "templates",
		DeviceMappings: []common.DeviceTemplateMapping{
			{DeviceGUID: guid, DeviceName: "Arduino Due", TemplateFilepath: "custom/due.svg", DeviceTargetNumber: 1002},
		},
	}

	entry := buildDeviceEntry(config, &common.Device{GUID: guid, Name: "Arduino Due"},
		[]string{guid}, []string{"DCS World"}, &common.Profile{}, nil)

	if entry.TemplatePath != "custom/due.svg" {
		t.Errorf("templatePath = %q", entry.TemplatePath)
	}
	if entry.TemplateName != "due.svg" {
		t.Errorf("templateName = %q, want the basename", entry.TemplateName)
	}
	if entry.TargetNumber != 1002 {
		t.Errorf("targetNumber = %d, want 1002", entry.TargetNumber)
	}
}

func TestBuildDeviceEntry_ReportsSkippedDevice(t *testing.T) {
	const guid = "guid-skip"
	config := &common.Config{DeviceMappings: []common.DeviceTemplateMapping{
		{DeviceGUID: guid, DeviceName: "T-Rudder", SkipTemplate: true},
	}}

	entry := buildDeviceEntry(config, &common.Device{GUID: guid, Name: "T-Rudder"},
		[]string{guid}, nil, &common.Profile{}, nil)

	if !entry.Skipped {
		t.Error("skipped = false, want true")
	}
}

// --- routes

func deviceTestState(t *testing.T) (*State, string) {
	t.Helper()

	templates := t.TempDir()
	if err := os.WriteFile(filepath.Join(templates, "stick.svg"), []byte("<svg>Button_1</svg>"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	return newTestState(t, &common.Config{TemplatesDirectory: templates}), templates
}

// Modules are detected, not recorded. The routes that used to write them are
// gone, and nothing should answer at their addresses.
func TestDeviceRoute_NoModuleRoutes(t *testing.T) {
	h := NewHandler(newTestState(t, &common.Config{}))

	for _, path := range []string{"/api/devices/modules", "/api/devices/duplicate-module"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}")))

		if rec.Code != http.StatusNotFound {
			t.Errorf("POST %s = %d, want 404", path, rec.Code)
		}
	}
}

func TestDeviceRoute_AssignPersistsMapping(t *testing.T) {
	state, _ := deviceTestState(t)
	h := NewHandler(state)

	body, _ := json.Marshal(mappingRequest{
		Action: "assign", GUID: "guid-a", Name: "Test Stick", TemplatePath: "stick.svg",
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/devices/mapping", strings.NewReader(string(body))))

	if rec.Code != http.StatusOK {
		t.Fatalf("assign = %d (%s), want 200", rec.Code, rec.Body.String())
	}

	written, err := common.LoadConfigFrom(state.ConfigPath())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	mapping := written.GetTemplateMappingForDevice("guid-a")
	if mapping == nil || mapping.TemplateFilepath != "stick.svg" {
		t.Errorf("persisted mapping = %+v, want stick.svg", mapping)
	}
}

func TestDeviceRoute_SkipPersists(t *testing.T) {
	state, _ := deviceTestState(t)
	h := NewHandler(state)

	body, _ := json.Marshal(mappingRequest{Action: "skip", GUID: "guid-b", Name: "T-Rudder"})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/devices/mapping", strings.NewReader(string(body))))

	if rec.Code != http.StatusOK {
		t.Fatalf("skip = %d (%s), want 200", rec.Code, rec.Body.String())
	}

	written, _ := common.LoadConfigFrom(state.ConfigPath())
	mapping := written.GetTemplateMappingForDevice("guid-b")
	if mapping == nil || !mapping.SkipTemplate {
		t.Errorf("persisted mapping = %+v, want skip_template true", mapping)
	}
}

// A template path is written straight into the configuration. It must be
// rejected when it points outside the templates directory.
func TestDeviceRoute_RejectsTemplateOutsideDirectory(t *testing.T) {
	state, _ := deviceTestState(t)
	h := NewHandler(state)

	body, _ := json.Marshal(mappingRequest{
		Action: "assign", GUID: "guid-a", Name: "Test", TemplatePath: "../../../etc/passwd",
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/devices/mapping", strings.NewReader(string(body))))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("assign outside the templates directory = %d, want 400", rec.Code)
	}
}

func TestDeviceRoute_RejectsUnknownAction(t *testing.T) {
	state, _ := deviceTestState(t)
	h := NewHandler(state)

	body, _ := json.Marshal(mappingRequest{Action: "delete", GUID: "guid-a"})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/devices/mapping", strings.NewReader(string(body))))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("unknown action = %d, want 400", rec.Code)
	}
}

func TestDeviceRoute_RequiresGUID(t *testing.T) {
	state, _ := deviceTestState(t)
	h := NewHandler(state)

	body, _ := json.Marshal(mappingRequest{Action: "assign", TemplatePath: "stick.svg"})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/devices/mapping", strings.NewReader(string(body))))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("assign without a guid = %d, want 400", rec.Code)
	}
}

// Scanning with nothing configured must return an empty list, not fail.
func TestScanDevices_NoSimulatorsConfigured(t *testing.T) {
	payload := scanDevices(&common.Config{})

	if len(payload.Devices) != 0 {
		t.Errorf("devices = %d, want 0", len(payload.Devices))
	}
	if len(payload.Warnings) == 0 {
		t.Error("want a warning about the missing templates directory")
	}
}

// Nothing configured means nothing to detect, and the button must say so rather
// than silently doing nothing.
func TestTargetDetect_RefusesWithoutATargetProfile(t *testing.T) {
	state, _ := deviceTestState(t)

	rec := httptest.NewRecorder()
	NewHandler(state).ServeHTTP(rec,
		httptest.NewRequest(http.MethodPost, "/api/devices/target/detect", strings.NewReader("{}")))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("detect with no TARGET profile = %d, want 400", rec.Code)
	}
}

// The auto-matching this route exposes is the one thing the deleted command line
// walk did that the interface could not: matching device numbers to controllers
// by name, so nobody has to know that a Warthog stick is device 1001.
func TestDetectTargetNumbers_MatchesByName(t *testing.T) {
	numbers := []int{1001, 1002}
	devices := []*common.Device{
		{GUID: "{AAAA0000-0000-0000-0000-000000000000}", Name: "WINWING Orion Throttle Base II"},
		{GUID: "{BBBB0000-0000-0000-0000-000000000000}", Name: "WINWING Orion Joystick Base 2"},
		{GUID: "{CCCC0000-0000-0000-0000-000000000000}", Name: "Arduino Due"},
	}

	mappings := target.AutoMatchTargetDevices(numbers, common.FilterPhysicalDevices(devices))

	byNumber := map[int]string{}
	for _, m := range mappings {
		byNumber[m.DeviceNumber] = m.DeviceName
	}

	if got := byNumber[1001]; got != "WINWING Orion Joystick Base 2" {
		t.Errorf("1001 matched %q, want the joystick", got)
	}
	if got := byNumber[1002]; got != "WINWING Orion Throttle Base II" {
		t.Errorf("1002 matched %q, want the throttle", got)
	}
	if len(mappings) != 2 {
		t.Errorf("matched %d controllers, want 2, the Arduino matches no pattern", len(mappings))
	}
}
