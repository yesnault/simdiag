package svg

import (
	"strings"
	"testing"
	"time"

	"simdiag/common"
)

const testGUID = "{EE6F1C30-3F2E-11F0-8001-444553540000}"
const otherGUID = "{B0C891C0-3F30-11F0-8003-444553540000}"

// newTestDevice builds a minimal ExportDevice around a raw template body.
func newTestDevice(rawData string, bindings []common.Binding) *common.ExportDevice {
	return &common.ExportDevice{
		Device: &common.Device{Name: "Test Device", GUID: testGUID},
		Template: &common.Template{
			Name:     "test-template",
			FilePath: "test-template.svg",
			RawData:  rawData,
		},
		Profile: &common.Profile{
			Name:     "Test Profile",
			SimType:  common.DCSWorld,
			Devices:  map[string]*common.Device{testGUID: {Name: "Test Device", GUID: testGUID}},
			Bindings: bindings,
		},
	}
}

func buttonBinding(inputID, action string) common.Binding {
	return common.Binding{
		DeviceGUID: testGUID,
		DeviceName: "Test Device",
		InputType:  common.Button,
		InputID:    inputID,
		Action:     action,
	}
}

func mustRender(t *testing.T, d *common.ExportDevice) string {
	t.Helper()
	out, err := RenderSVG(d)
	if err != nil {
		t.Fatalf("RenderSVG() unexpected error: %v", err)
	}
	return out
}

func TestRenderSVG_NoTemplate(t *testing.T) {
	d := newTestDevice("<svg>Button_1</svg>", nil)
	d.Template = nil

	if _, err := RenderSVG(d); err == nil {
		t.Error("RenderSVG() with no template should return an error")
	}
}

func TestRenderSVG_NoProfile(t *testing.T) {
	d := newTestDevice("<svg>Button_1</svg>", nil)
	d.Profile = nil

	if _, err := RenderSVG(d); err == nil {
		t.Error("RenderSVG() with no profile should return an error")
	}
}

func TestRenderSVG_NilDevice(t *testing.T) {
	if _, err := RenderSVG(nil); err == nil {
		t.Error("RenderSVG(nil) should return an error")
	}
}

// The draw.io templates carry HTML labels double-escaped inside XML attributes,
// so a substituted action must come out as &amp;lt;div&amp;gt;...
func TestRenderSVG_ButtonBindingIsDoubleEscaped(t *testing.T) {
	d := newTestDevice("<svg>Button_1</svg>", []common.Binding{
		buttonBinding("1", "Master Arm"),
	})

	got := mustRender(t, d)

	want := "&amp;lt;div&amp;gt;Master Arm&amp;lt;/div&amp;gt;"
	if !strings.Contains(got, want) {
		t.Errorf("RenderSVG() = %q, want it to contain %q", got, want)
	}
	if strings.Contains(got, "Button_1") {
		t.Errorf("RenderSVG() = %q, should not still contain the raw key Button_1", got)
	}
}

func TestRenderSVG_UnusedKeysAreBlanked(t *testing.T) {
	d := newTestDevice("<svg>Button_1|Button_2|POV_1_U|AXIS_X</svg>", []common.Binding{
		buttonBinding("1", "Master Arm"),
	})

	got := mustRender(t, d)

	for _, key := range []string{"Button_2", "POV_1_U", "AXIS_X"} {
		if strings.Contains(got, key) {
			t.Errorf("RenderSVG() = %q, unused key %q should have been blanked", got, key)
		}
	}
}

// Anything the template loader accepts as a key has to be erased when nothing
// binds it, or the raw key is what the user reads on the diagram.
//
// The two lists were separate once, and the axis pattern was spelled Axis_ here
// and AXIS_ in common. Since every shipped template writes AXIS_X, no axis was
// ever blanked and the placeholders reached the finished PNG.
func TestRenderSVG_BlanksEveryKeyTheLoaderRecognises(t *testing.T) {
	keys := []string{"Button_7", "AXIS_X", "AXIS_SLIDER", "AXIS_RZ", "POV_1_U", "POV_2_D"}

	template := "<svg>" + strings.Join(keys, "|") + "</svg>"
	got := mustRender(t, newTestDevice(template, nil))

	for _, key := range keys {
		if !common.HasTemplateKey(key) {
			t.Errorf("the loader does not recognise %q, so this test proves nothing about it", key)
			continue
		}
		if strings.Contains(got, key) {
			t.Errorf("%q survived rendering with no binding on it", key)
		}
	}
}

// A binding that belongs to a different physical device must not leak into this
// device's diagram.
func TestRenderSVG_IgnoresBindingsFromOtherDevices(t *testing.T) {
	foreign := buttonBinding("1", "Wrong Device Action")
	foreign.DeviceGUID = otherGUID

	d := newTestDevice("<svg>Button_1</svg>", []common.Binding{foreign})

	got := mustRender(t, d)

	if strings.Contains(got, "Wrong Device Action") {
		t.Errorf("RenderSVG() = %q, should not contain a binding from another device", got)
	}
}

func TestRenderSVG_MultipleBindingsOnSameKey(t *testing.T) {
	d := newTestDevice("<svg>Button_1</svg>", []common.Binding{
		buttonBinding("1", "First Action"),
		buttonBinding("1", "Second Action"),
	})

	got := mustRender(t, d)

	for _, action := range []string{"First Action", "Second Action"} {
		if !strings.Contains(got, action) {
			t.Errorf("RenderSVG() = %q, want it to contain %q", got, action)
		}
	}
}

// The same action bound twice must be rendered once, not duplicated.
func TestRenderSVG_DeduplicatesIdenticalActions(t *testing.T) {
	d := newTestDevice("<svg>Button_1</svg>", []common.Binding{
		buttonBinding("1", "Master Arm"),
		buttonBinding("1", "Master Arm"),
	})

	got := mustRender(t, d)

	if n := strings.Count(got, "Master Arm"); n != 1 {
		t.Errorf("RenderSVG() rendered %q %d times, want 1", "Master Arm", n)
	}
}

func TestRenderSVG_ModifierBindingIsColored(t *testing.T) {
	binding := buttonBinding("1", "Emergency Jettison")
	binding.Modifiers = []common.Modifier{{Keys: []string{"JOY_BTN24"}, Action: "Modifier"}}
	binding.ModifierNum = 1

	d := newTestDevice("<svg>Button_1</svg>", []common.Binding{binding})

	got := mustRender(t, d)

	if !strings.Contains(got, "font style=") || !strings.Contains(got, "color:") {
		t.Errorf("RenderSVG() = %q, want a coloured font span for a modifier binding", got)
	}
	if !strings.Contains(got, getModifierColor(1)) {
		t.Errorf("RenderSVG() = %q, want it to use the colour for modifier 1 (%s)", got, getModifierColor(1))
	}
}

func TestRenderSVG_ReplacesMetadata(t *testing.T) {
	d := newTestDevice("<svg>TEMPLATE_NAME|TITLE|SIMULATOR|SIMDIAG_VERSION|CURRENT_DATE</svg>", nil)
	d.Title = "DCS World / M-2000C"
	d.SimulatorName = "DCS World"
	d.SimdiagVersion = "v1.2.3"

	got := mustRender(t, d)

	for _, want := range []string{"Test Profile", "DCS World / M-2000C", "v1.2.3", time.Now().Format("2006-01-02")} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderSVG() = %q, want it to contain %q", got, want)
		}
	}
	for _, placeholder := range []string{"TEMPLATE_NAME", "SIMDIAG_VERSION", "CURRENT_DATE"} {
		if strings.Contains(got, placeholder) {
			t.Errorf("RenderSVG() = %q, placeholder %q should have been replaced", got, placeholder)
		}
	}
}

// An empty metadata value leaves the placeholder alone, so the template keeps
// whatever it already displays.
func TestRenderSVG_KeepsPlaceholderWhenMetadataEmpty(t *testing.T) {
	d := newTestDevice("<svg>TITLE</svg>", nil)
	d.Title = ""

	got := mustRender(t, d)

	if !strings.Contains(got, "TITLE") {
		t.Errorf("RenderSVG() = %q, want the TITLE placeholder kept when no title is set", got)
	}
}

func TestRenderSVG_SanitizesProfileName(t *testing.T) {
	d := newTestDevice("<svg>TEMPLATE_NAME</svg>", nil)
	d.Profile.Name = `A & B <script>`

	got := mustRender(t, d)

	if strings.Contains(got, "<script>") {
		t.Errorf("RenderSVG() = %q, must not emit a raw <script> tag", got)
	}
}

// RenderSVG must not touch the filesystem, and must leave the template untouched
// so the same *common.Template can be reused across devices.
func TestRenderSVG_DoesNotMutateTemplate(t *testing.T) {
	raw := "<svg>Button_1|Button_2</svg>"
	d := newTestDevice(raw, []common.Binding{buttonBinding("1", "Master Arm")})

	mustRender(t, d)

	if d.Template.RawData != raw {
		t.Errorf("RenderSVG() mutated Template.RawData: got %q, want %q", d.Template.RawData, raw)
	}
}
