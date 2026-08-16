package workflow

import (
	"strings"
	"testing"

	"simdiag/common"
)

// The rule itself lives in common.SimulatorIsConfigured and is tested there; what
// matters here is how the export reacts to it.

// TestShouldProcessSimulator_WarnsOnlyWhenPartlyConfigured checks that a
// simulator the user never set up stays quiet, while one that has a section but
// lacks its path says so. The skip used to be silent in both cases.
func TestShouldProcessSimulator_WarnsOnlyWhenPartlyConfigured(t *testing.T) {
	base := func(dcs *common.SimulatorConfig) *common.Config {
		config := &common.Config{
			TemplatesDirectory: "templates",
			OutputDirectory:    "output",
			Simulators:         map[string]*common.SimulatorConfig{},
		}
		if dcs != nil {
			config.Simulators["dcs_world"] = dcs
		}
		return config
	}

	t.Run("no DCS section at all", func(t *testing.T) {
		summary := &ExportSummary{}
		if shouldProcessSimulator(common.DCSWorld, base(nil), "", summary) {
			t.Error("DCS should not be processed without a section")
		}
		if len(summary.Warnings) != 0 {
			t.Errorf("an unused simulator should not warn: %v", summary.Warnings)
		}
	})

	t.Run("DCS section without a path", func(t *testing.T) {
		summary := &ExportSummary{}
		if shouldProcessSimulator(common.DCSWorld, base(&common.SimulatorConfig{GremlinsProfileFilepath: `C:\gremlins.xml`}), "", summary) {
			t.Error("DCS should not be processed without dcs_path")
		}
		if len(summary.Warnings) != 1 {
			t.Fatalf("expected one warning, got %v", summary.Warnings)
		}
		if !strings.Contains(summary.Warnings[0], "dcs_path") {
			t.Errorf("the warning should name the missing setting: %q", summary.Warnings[0])
		}
	})

	t.Run("DCS section with a path", func(t *testing.T) {
		summary := &ExportSummary{}
		if !shouldProcessSimulator(common.DCSWorld, base(&common.SimulatorConfig{DCSPath: `C:\DCS`}), "", summary) {
			t.Errorf("DCS with a path should be processed, warnings: %v", summary.Warnings)
		}
	})
}

// TestShouldProcessSimulator_DoesNotCreateSections pins that asking whether a
// simulator should run leaves the configuration alone. It runs for every
// simulator of every export, and used to insert an empty section as a side
// effect, which then got written back to disk.
func TestShouldProcessSimulator_DoesNotCreateSections(t *testing.T) {
	config := &common.Config{
		TemplatesDirectory: "templates",
		OutputDirectory:    "output",
		Simulators:         map[string]*common.SimulatorConfig{},
	}

	shouldProcessSimulator(common.IL2Korea, config, "", &ExportSummary{})

	if len(config.Simulators) != 0 {
		t.Errorf("configuration grew a section: %+v", config.Simulators)
	}
}

// TestMatchesModuleFilter pins the two namespaces a DCS module filter can be
// written in: the raw profile folder name ("M-2000C", what the CLI user types)
// and the normalized config key ("m2000c", what the GUI export dropdown sends).
func TestMatchesModuleFilter(t *testing.T) {
	tests := []struct {
		name     string
		module   string
		filter   string
		expected bool
	}{
		{"normalized key from the GUI dropdown", "M-2000C", "m2000c", true},
		{"partial raw name from the CLI", "M-2000C", "2000", true},
		{"exact raw name", "M-2000C", "M-2000C", true},
		{"case-insensitive raw name", "M-2000C", "m-2000c", true},
		{"empty filter matches everything", "M-2000C", "", true},
		{"normalized key with suffix stripped", "FA-18C_hornet", "fa18c", true},
		{"partial raw name with underscore", "FA-18C_hornet", "hornet", true},
		{"normalized key with variant stripped", "P-47D-30", "p47d", true},
		{"another module does not match", "M-2000C", "fa18c", false},
		{"unknown filter matches nothing", "M-2000C", "viggen", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchesModuleFilter(tt.module, tt.filter); got != tt.expected {
				t.Errorf("MatchesModuleFilter(%q, %q) = %v, want %v",
					tt.module, tt.filter, got, tt.expected)
			}
		})
	}
}
