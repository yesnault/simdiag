package common

import "testing"

func TestFormatInputText(t *testing.T) {
	tests := []struct {
		inputType InputType
		inputID   string
		want      string
	}{
		{Button, "1", "BTN1"},
		{Button, "105", "BTN105"},
		{Axis, "X", "Axis X"},
		{Axis, "SLIDER_1", "Axis SLIDER_1"},
		{Hat, "1_U", "POV 1_U"},
		{InputType("unknown"), "1", ""},
	}

	for _, tt := range tests {
		if got := FormatInputText(tt.inputType, tt.inputID); got != tt.want {
			t.Errorf("FormatInputText(%q, %q) = %q, want %q", tt.inputType, tt.inputID, got, tt.want)
		}
	}
}

func TestParseInputText(t *testing.T) {
	tests := []struct {
		text          string
		wantType      InputType
		wantID        string
		wantRecognise bool
	}{
		{"BTN1", Button, "1", true},
		{"Axis X", Axis, "X", true},
		{"POV 1_U", Hat, "1_U", true},
		{"", "", "", false},
		{"Keyboard F1", "", "", false},
	}

	for _, tt := range tests {
		gotType, gotID, gotOK := ParseInputText(tt.text)
		if gotType != tt.wantType || gotID != tt.wantID || gotOK != tt.wantRecognise {
			t.Errorf("ParseInputText(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.text, gotType, gotID, gotOK, tt.wantType, tt.wantID, tt.wantRecognise)
		}
	}
}

// TestInputTextRoundTrip guards the CSV -> SVG round trip: every input written by
// the exporter must be readable back by the importer without loss.
func TestInputTextRoundTrip(t *testing.T) {
	cases := []struct {
		inputType InputType
		inputID   string
	}{
		{Button, "1"},
		{Button, "105"},
		{Button, "25_OFF"},
		{Axis, "X"},
		{Axis, "RZ"},
		{Axis, "SLIDER_1"},
		{Hat, "1_U"},
		{Hat, "2_DL"},
	}

	for _, c := range cases {
		text := FormatInputText(c.inputType, c.inputID)

		gotType, gotID, ok := ParseInputText(text)
		if !ok {
			t.Errorf("ParseInputText(%q) failed to recognise output of FormatInputText(%q, %q)", text, c.inputType, c.inputID)
			continue
		}
		if gotType != c.inputType || gotID != c.inputID {
			t.Errorf("round trip of (%q, %q) via %q gave (%q, %q)", c.inputType, c.inputID, text, gotType, gotID)
		}
	}
}

func TestTemplateKeyFor(t *testing.T) {
	tests := []struct {
		inputType InputType
		inputID   string
		want      string
	}{
		{Button, "1", "Button_1"},
		{Button, "25", "Button_25"},
		{Button, "25_OFF", "Button_25"}, // _OFF collapses onto the base key
		{Axis, "x", "AXIS_X"},
		{Axis, "slider_1", "AXIS_SLIDER_1"},
		{Hat, "1_u", "POV_1_U"},
		{InputType("unknown"), "1", ""},
	}

	for _, tt := range tests {
		if got := TemplateKeyFor(tt.inputType, tt.inputID); got != tt.want {
			t.Errorf("TemplateKeyFor(%q, %q) = %q, want %q", tt.inputType, tt.inputID, got, tt.want)
		}
	}
}
