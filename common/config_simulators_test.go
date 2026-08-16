package common

import "testing"

// TestLookupSimulatorConfig separates "never configured" from "configured
// incompletely": the export warns about the second but not the first.
func TestLookupSimulatorConfig(t *testing.T) {
	config := &Config{
		Simulators: map[string]*SimulatorConfig{"dcs_world": {DCSPath: `C:\DCS`}},
	}

	if section := config.LookupSimulatorConfig(DCSWorld); section == nil || section.DCSPath != `C:\DCS` {
		t.Errorf("dcs_world should resolve, got %+v", section)
	}
	if section := config.LookupSimulatorConfig(IL2Korea); section != nil {
		t.Errorf("an absent simulator should return nil, got %+v", section)
	}
	if len(config.Simulators) != 1 {
		t.Errorf("looking a simulator up created it: %v", config.Simulators)
	}

	var nilConfig *Config
	if section := nilConfig.LookupSimulatorConfig(DCSWorld); section != nil {
		t.Errorf("a nil config should return nil, got %+v", section)
	}
}

// TestSimulatorIsConfigured pins the single rule both the batch validation and
// the export gate ask. DCS hangs on its path: its aircraft are detected from
// that path, never declared in the configuration.
func TestSimulatorIsConfigured(t *testing.T) {
	tests := []struct {
		name      string
		simType   SimulationType
		simConfig *SimulatorConfig
		expected  bool
	}{
		{"DCS with a path", DCSWorld, &SimulatorConfig{DCSPath: `C:\DCS`}, true},
		{"DCS without one", DCSWorld, &SimulatorConfig{GremlinsProfileFilepath: `C:\gremlins.xml`}, false},
		{"IL-2 with an input path", IL2Sturmovik, &SimulatorConfig{IL2InputPath: `C:\IL-2`}, true},
		{"IL-2 without one", IL2Sturmovik, &SimulatorConfig{GremlinsProfileFilepath: `C:\gremlins.xml`}, false},
		{"DCS path does not configure IL-2", IL2Sturmovik, &SimulatorConfig{DCSPath: `C:\DCS`}, false},
		{"nil section", DCSWorld, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SimulatorIsConfigured(tt.simType, tt.simConfig); got != tt.expected {
				t.Errorf("SimulatorIsConfigured(%s) = %v, want %v", tt.simType, got, tt.expected)
			}
		})
	}
}

// Callers that iterate Config.Simulators hold a key, not a type.
func TestSimulationTypeForConfigKey(t *testing.T) {
	for _, simType := range []SimulationType{DCSWorld, IL2Sturmovik, IL2Korea} {
		if got := SimulationTypeForConfigKey(simType.GetConfigKey()); got != simType {
			t.Errorf("SimulationTypeForConfigKey(%q) = %q, want %q", simType.GetConfigKey(), got, simType)
		}
	}
	if got := SimulationTypeForConfigKey("something_else"); got != "" {
		t.Errorf("an unknown key should yield the empty type, got %q", got)
	}
}
