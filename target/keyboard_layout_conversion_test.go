package target

import "testing"

// TestConvertKeyForLayout tests keyboard layout conversion
func TestConvertKeyForLayout(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		sourceLayout KeyboardLayout
		targetLayout KeyboardLayout
		want         string
	}{
		// Same layout - no conversion
		{
			name:         "same layout (QWERTY)",
			key:          "A",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardQWERTY,
			want:         "A",
		},
		{
			name:         "same layout (AZERTY)",
			key:          "A",
			sourceLayout: KeyboardAZERTY,
			targetLayout: KeyboardAZERTY,
			want:         "A",
		},
		// QWERTY to AZERTY
		{
			name:         "QWERTY Q -> AZERTY A",
			key:          "Q",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         "A",
		},
		{
			name:         "QWERTY W -> AZERTY Z",
			key:          "W",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         "Z",
		},
		{
			name:         "QWERTY A -> AZERTY Q",
			key:          "A",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         "Q",
		},
		{
			name:         "QWERTY Z -> AZERTY W",
			key:          "Z",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         "W",
		},
		{
			name:         "QWERTY M -> AZERTY ,",
			key:          "M",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         ",",
		},
		{
			name:         "QWERTY ; -> AZERTY m",
			key:          ";",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         "m",
		},
		{
			name:         "QWERTY = -> AZERTY )",
			key:          "=",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         ")",
		},
		{
			name:         "QWERTY - -> AZERTY )",
			key:          "-",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         ")",
		},
		// AZERTY to QWERTY
		{
			name:         "AZERTY A -> QWERTY Q",
			key:          "A",
			sourceLayout: KeyboardAZERTY,
			targetLayout: KeyboardQWERTY,
			want:         "Q",
		},
		{
			name:         "AZERTY Z -> QWERTY W",
			key:          "Z",
			sourceLayout: KeyboardAZERTY,
			targetLayout: KeyboardQWERTY,
			want:         "W",
		},
		{
			name:         "AZERTY Q -> QWERTY A",
			key:          "Q",
			sourceLayout: KeyboardAZERTY,
			targetLayout: KeyboardQWERTY,
			want:         "A",
		},
		{
			name:         "AZERTY W -> QWERTY Z",
			key:          "W",
			sourceLayout: KeyboardAZERTY,
			targetLayout: KeyboardQWERTY,
			want:         "Z",
		},
		{
			name:         "AZERTY ) -> QWERTY -",
			key:          ")",
			sourceLayout: KeyboardAZERTY,
			targetLayout: KeyboardQWERTY,
			want:         "-",
		},
		{
			name:         "AZERTY & -> QWERTY 1",
			key:          "&",
			sourceLayout: KeyboardAZERTY,
			targetLayout: KeyboardQWERTY,
			want:         "1",
		},
		{
			name:         "AZERTY é -> QWERTY 2",
			key:          "é",
			sourceLayout: KeyboardAZERTY,
			targetLayout: KeyboardQWERTY,
			want:         "2",
		},
		// Case preservation
		{
			name:         "lowercase conversion",
			key:          "q",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         "a",
		},
		{
			name:         "uppercase conversion",
			key:          "Q",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         "A",
		},
		// Unmapped keys
		{
			name:         "unmapped key (B)",
			key:          "B",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         "B",
		},
		{
			name:         "unmapped key (1)",
			key:          "1",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         "1",
		},
		{
			name:         "unmapped special key (Space)",
			key:          "Space",
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         "Space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertKeyForLayout(tt.key, tt.sourceLayout, tt.targetLayout)
			if got != tt.want {
				t.Errorf("ConvertKeyForLayout(%q, %v, %v) = %q, want %q", tt.key, tt.sourceLayout, tt.targetLayout, got, tt.want)
			}
		})
	}
}

// TestConvertKeysForLayout tests batch keyboard layout conversion
func TestConvertKeysForLayout(t *testing.T) {
	tests := []struct {
		name         string
		keys         []string
		sourceLayout KeyboardLayout
		targetLayout KeyboardLayout
		want         []string
	}{
		{
			name:         "QWERTY to AZERTY - multiple keys",
			keys:         []string{"Q", "W", "A", "Z"},
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         []string{"A", "Z", "Q", "W"},
		},
		{
			name:         "same layout - no change",
			keys:         []string{"Q", "W", "E", "R"},
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardQWERTY,
			want:         []string{"Q", "W", "E", "R"},
		},
		{
			name:         "empty slice",
			keys:         []string{},
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         []string{},
		},
		{
			name:         "mixed mapped and unmapped",
			keys:         []string{"Q", "B", "W", "C"},
			sourceLayout: KeyboardQWERTY,
			targetLayout: KeyboardAZERTY,
			want:         []string{"A", "B", "Z", "C"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertKeysForLayout(tt.keys, tt.sourceLayout, tt.targetLayout)
			if len(got) != len(tt.want) {
				t.Errorf("ConvertKeysForLayout() returned %d keys, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ConvertKeysForLayout() key[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
