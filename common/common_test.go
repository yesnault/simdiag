package common

import (
	"testing"
)

// TestNormalizeGUID tests GUID normalization to lowercase without braces
func TestNormalizeGUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "uppercase with braces",
			input: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			want:  "ee6f1c30-3f2e-11f0-8001-444553540000",
		},
		{
			name:  "lowercase with braces",
			input: "{ee6f1c30-3f2e-11f0-8001-444553540000}",
			want:  "ee6f1c30-3f2e-11f0-8001-444553540000",
		},
		{
			name:  "mixed case with braces",
			input: "{Ee6F1c30-3f2E-11F0-8001-444553540000}",
			want:  "ee6f1c30-3f2e-11f0-8001-444553540000",
		},
		{
			name:  "uppercase without braces",
			input: "EE6F1C30-3F2E-11F0-8001-444553540000",
			want:  "ee6f1c30-3f2e-11f0-8001-444553540000",
		},
		{
			name:  "lowercase without braces",
			input: "ee6f1c30-3f2e-11f0-8001-444553540000",
			want:  "ee6f1c30-3f2e-11f0-8001-444553540000",
		},
		{
			name:  "IL-2 4-segment GUID",
			input: "{A7C91C00-3F30-11F0-0000545345440180}",
			want:  "a7c91c00-3f30-11f0-0000545345440180",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only braces",
			input: "{}",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeGUID(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeGUID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestNormalizeGUIDUpper tests GUID normalization to uppercase without braces
func TestNormalizeGUIDUpper(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase with braces",
			input: "{ee6f1c30-3f2e-11f0-8001-444553540000}",
			want:  "EE6F1C30-3F2E-11F0-8001-444553540000",
		},
		{
			name:  "uppercase with braces",
			input: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			want:  "EE6F1C30-3F2E-11F0-8001-444553540000",
		},
		{
			name:  "mixed case without braces",
			input: "Ee6F1c30-3f2E-11F0-8001-444553540000",
			want:  "EE6F1C30-3F2E-11F0-8001-444553540000",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeGUIDUpper(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeGUIDUpper(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestNormalizeGUIDBraced tests GUID normalization to uppercase with braces
func TestNormalizeGUIDBraced(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase without braces",
			input: "ee6f1c30-3f2e-11f0-8001-444553540000",
			want:  "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		{
			name:  "uppercase without braces",
			input: "EE6F1C30-3F2E-11F0-8001-444553540000",
			want:  "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		{
			name:  "already has braces",
			input: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			want:  "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		{
			name:  "has left brace only",
			input: "{EE6F1C30-3F2E-11F0-8001-444553540000",
			want:  "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		{
			name:  "has right brace only",
			input: "EE6F1C30-3F2E-11F0-8001-444553540000}",
			want:  "{EE6F1C30-3F2E-11F0-8001-444553540000}",
		},
		{
			name:  "empty string",
			input: "",
			want:  "{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeGUIDBraced(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeGUIDBraced(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestNormalizeGUIDShort tests GUID normalization to first 8 characters
func TestNormalizeGUIDShort(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full GUID with braces",
			input: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			want:  "ee6f1c30",
		},
		{
			name:  "full GUID without braces",
			input: "EE6F1C30-3F2E-11F0-8001-444553540000",
			want:  "ee6f1c30",
		},
		{
			name:  "short GUID",
			input: "ABC",
			want:  "abc",
		},
		{
			name:  "exactly 8 chars",
			input: "12345678",
			want:  "12345678",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeGUIDShort(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeGUIDShort(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestMatchGUIDPartial tests partial GUID matching for cross-simulator compatibility
func TestMatchGUIDPartial(t *testing.T) {
	tests := []struct {
		name  string
		guid1 string
		guid2 string
		want  bool
	}{
		{
			name:  "exact match - both 5 segments",
			guid1: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			guid2: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			want:  true,
		},
		{
			name:  "exact match - both 4 segments",
			guid1: "{A7C91C00-3F30-11F0-0000545345440180}",
			guid2: "{A7C91C00-3F30-11F0-0000545345440180}",
			want:  true,
		},
		{
			name:  "partial match - 5 vs 4 segments (DCS vs IL-2)",
			guid1: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			guid2: "{EE6F1C30-3F2E-11F0-0000545345440180}",
			want:  true,
		},
		{
			name:  "partial match - 4 vs 5 segments (IL-2 vs DCS)",
			guid1: "{A7C91C00-3F30-11F0-0000545345440180}",
			guid2: "{A7C91C00-3F30-11F0-8001-444553540000}",
			want:  true,
		},
		{
			name:  "no match - different first segment",
			guid1: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			guid2: "{A7C91C00-3F2E-11F0-8001-444553540000}",
			want:  false,
		},
		{
			name:  "no match - different second segment",
			guid1: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			guid2: "{EE6F1C30-FFFF-11F0-8001-444553540000}",
			want:  false,
		},
		{
			name:  "no match - different third segment",
			guid1: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			guid2: "{EE6F1C30-3F2E-FFFF-8001-444553540000}",
			want:  false,
		},
		{
			name:  "no match - same format but different 4th/5th segments",
			guid1: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			guid2: "{EE6F1C30-3F2E-11F0-FFFF-FFFFFFFFFFFF}",
			want:  false,
		},
		{
			name:  "case insensitive",
			guid1: "{ee6f1c30-3f2e-11f0-8001-444553540000}",
			guid2: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			want:  true,
		},
		{
			name:  "with and without braces",
			guid1: "EE6F1C30-3F2E-11F0-8001-444553540000",
			guid2: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			want:  true,
		},
		{
			name:  "invalid - too few segments",
			guid1: "EE6F1C30-3F2E",
			guid2: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			want:  false,
		},
		{
			name:  "empty strings",
			guid1: "",
			guid2: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchGUIDPartial(tt.guid1, tt.guid2)
			if got != tt.want {
				t.Errorf("MatchGUIDPartial(%q, %q) = %v, want %v", tt.guid1, tt.guid2, got, tt.want)
			}
		})
	}
}

// TestIsVirtualDevice tests virtual device detection by name
func TestIsVirtualDevice(t *testing.T) {
	tests := []struct {
		name       string
		deviceName string
		want       bool
	}{
		{
			name:       "vJoy device",
			deviceName: "vJoy Device 1",
			want:       true,
		},
		{
			name:       "vJoy lowercase",
			deviceName: "vjoy device 2",
			want:       true,
		},
		{
			name:       "vJoy uppercase",
			deviceName: "VJOY DEVICE 3",
			want:       true,
		},
		{
			name:       "generic virtual",
			deviceName: "Virtual Joystick",
			want:       true,
		},
		{
			name:       "TARGET combined device",
			deviceName: "Thrustmaster Combined",
			want:       true,
		},
		{
			name:       "combined keyword",
			deviceName: "Combined Device",
			want:       true,
		},
		{
			name:       "physical device - Warthog Joystick",
			deviceName: "Joystick - HOTAS Warthog",
			want:       false,
		},
		{
			name:       "physical device - Warthog Throttle",
			deviceName: "Throttle - HOTAS Warthog",
			want:       false,
		},
		{
			name:       "physical device - VKB Gladiator",
			deviceName: "VKB-Sim Gladiator EVO",
			want:       false,
		},
		{
			name:       "physical device - Virpil",
			deviceName: "VPC MongoosT-50CM2 Throttle",
			want:       false,
		},
		{
			name:       "empty string",
			deviceName: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVirtualDevice(tt.deviceName)
			if got != tt.want {
				t.Errorf("IsVirtualDevice(%q) = %v, want %v", tt.deviceName, got, tt.want)
			}
		})
	}
}

// TestIsVirtualDeviceGUID tests virtual device detection by GUID
func TestIsVirtualDeviceGUID(t *testing.T) {
	tests := []struct {
		name       string
		deviceGUID string
		want       bool
	}{
		{
			name:       "vJoy device with b302-11ea pattern",
			deviceGUID: "{A768EF40-B302-11EA-8001-444553540000}",
			want:       true,
		},
		{
			name:       "vJoy device with b310-11ea pattern",
			deviceGUID: "{A768EF40-B310-11EA-8001-444553540000}",
			want:       true,
		},
		{
			name:       "vJoy device lowercase",
			deviceGUID: "{a768ef40-b302-11ea-8001-444553540000}",
			want:       true,
		},
		{
			name:       "physical device - Warthog",
			deviceGUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}",
			want:       false,
		},
		{
			name:       "physical device - IL-2 format",
			deviceGUID: "{A7C91C00-3F30-11F0-0000545345440180}",
			want:       false,
		},
		{
			name:       "empty string",
			deviceGUID: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVirtualDeviceGUID(tt.deviceGUID)
			if got != tt.want {
				t.Errorf("IsVirtualDeviceGUID(%q) = %v, want %v", tt.deviceGUID, got, tt.want)
			}
		})
	}
}

// TestFilterPhysicalDevices tests filtering of physical devices
func TestFilterPhysicalDevices(t *testing.T) {
	tests := []struct {
		name    string
		devices []*Device
		want    int // number of physical devices expected
	}{
		{
			name: "mix of physical and virtual devices",
			devices: []*Device{
				{Name: "Joystick - HOTAS Warthog", GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}", IsVirtual: false},
				{Name: "vJoy Device 1", GUID: "{A768EF40-B302-11EA-8001-444553540000}", IsVirtual: true},
				{Name: "Throttle - HOTAS Warthog", GUID: "{A7C91C00-3F30-11F0-8001-444553540000}", IsVirtual: false},
				{Name: "Thrustmaster Combined", GUID: "{12345678-1234-1234-1234-123456789012}", IsVirtual: true},
			},
			want: 2,
		},
		{
			name: "all physical devices",
			devices: []*Device{
				{Name: "Joystick", GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}", IsVirtual: false},
				{Name: "Throttle", GUID: "{A7C91C00-3F30-11F0-8001-444553540000}", IsVirtual: false},
			},
			want: 2,
		},
		{
			name: "all virtual devices",
			devices: []*Device{
				{Name: "vJoy Device 1", GUID: "{A768EF40-B302-11EA-8001-444553540000}", IsVirtual: true},
				{Name: "vJoy Device 2", GUID: "{A768EF40-B310-11EA-8001-444553540000}", IsVirtual: true},
			},
			want: 0,
		},
		{
			name:    "empty slice",
			devices: []*Device{},
			want:    0,
		},
		{
			name: "nil devices in slice",
			devices: []*Device{
				{Name: "Joystick", GUID: "{EE6F1C30-3F2E-11F0-8001-444553540000}", IsVirtual: false},
				nil,
				{Name: "Throttle", GUID: "{A7C91C00-3F30-11F0-8001-444553540000}", IsVirtual: false},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterPhysicalDevices(tt.devices)
			if len(got) != tt.want {
				t.Errorf("FilterPhysicalDevices() returned %d devices, want %d", len(got), tt.want)
			}
			// Verify all returned devices are physical
			for _, device := range got {
				if device.IsVirtual {
					t.Errorf("FilterPhysicalDevices() returned virtual device %q", device.Name)
				}
			}
		})
	}
}
