package common

import (
	"os"
	"path/filepath"
)

// Where a simulator keeps the controller configuration SimDiag reads.
//
// Nothing here prompts. Configuration is the graphical interface's job. These
// two functions answer "where is it" for the export, and fill the suggestion the
// Configuration tab offers for an empty field.

// DefaultSimulatorPath returns the stock install location of a simulator's
// configuration files, used when nothing is configured yet.
func DefaultSimulatorPath(simType SimulationType) string {
	switch simType {
	case DCSWorld:
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, "Saved Games", "DCS")
	case IL2Sturmovik:
		return filepath.Join("C:\\", "Program Files", "IL-2 Sturmovik Great Battles", "data", "input")
	case IL2Korea:
		return filepath.Join("C:\\", "Program Files", "IL2Series", "game", "data", "Input")
	}
	return ""
}

// ConfiguredSimulatorPath returns the configured path for a simulator, falling
// back to DefaultSimulatorPath.
//
// The lookup does not create the section it reads. It used to, through
// EnsureSimulatorConfig, so asking where DCS was installed added an empty
// dcs_world entry that the next save wrote into the user's YAML.
func ConfiguredSimulatorPath(config *Config, simType SimulationType) string {
	if config != nil {
		simConfig := config.LookupSimulatorConfig(simType)
		if simConfig != nil {
			switch simType {
			case DCSWorld:
				if simConfig.DCSPath != "" {
					return simConfig.DCSPath
				}
			case IL2Sturmovik, IL2Korea:
				if simConfig.IL2InputPath != "" {
					return simConfig.IL2InputPath
				}
			}
		}
	}
	return DefaultSimulatorPath(simType)
}
