package srs

import (
	"fmt"
	"os"
	"strings"

	"simdiag/common"
)

// Configurator handles SRS configuration
type Configurator struct{}

// NewConfigurator creates a new SRS configurator
func NewConfigurator() *Configurator {
	return &Configurator{}
}

// GetName returns the name of the configurator
func (c *Configurator) GetName() string {
	return "Simple Radio Standalone (SRS)"
}

// configureSRSForSimulator configures SRS path for a specific simulator
func configureSRSForSimulator(config *common.Config, simType common.SimulationType, simName, dirName, foundPath string, found, batchMode bool) bool {
	needsSave := false

	if !found && !batchMode {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Printf("%s Simple Radio Standalone Configuration (Optional)\n", simName)
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println("\nSRS allows radio bindings integration.")
		fmt.Printf("Default path: C:\\Program Files\\%s\n", dirName)

		if common.AskYesNo(fmt.Sprintf("Do you want to configure %s SRS path?", simName)) {
			fmt.Printf("\nEnter the full path to %s directory: ", dirName)
			path, err := common.ReadLine()
			if err == nil && path != "" {
				path = strings.Trim(path, "\"'")

				if _, err := os.Stat(path); err == nil {
					simConfig := config.GetSimulatorConfig(simType)
					simConfig.SRSPath = path
					needsSave = true
					fmt.Printf("✓ %s SRS path saved: %s\n", simName, path)
				} else {
					fmt.Printf("⚠ Directory not found: %s\n", path)
				}
			}
		} else {
			fmt.Printf("⊘ %s SRS configuration skipped\n", simName)
		}
	} else if found {
		simConfig := config.GetSimulatorConfig(simType)
		if simConfig.SRSPath == "" {
			// Found at default location, save it
			simConfig.SRSPath = foundPath
			needsSave = true
		}
	}

	return needsSave
}

// Configure prompts the user to configure SRS paths for both simulators
func (c *Configurator) Configure(config *common.Config, batchMode bool) error {
	if config == nil {
		config = &common.Config{Simulators: make(map[string]*common.SimulatorConfig)}
	}

	il2Path, dcsPath, il2Found, dcsFound := common.VerifySRSPaths(config)

	needsSave := false
	needsSave = configureSRSForSimulator(config, common.IL2Sturmovik, "IL-2", "IL2-SimpleRadio-Standalone", il2Path, il2Found, batchMode) || needsSave
	needsSave = configureSRSForSimulator(config, common.DCSWorld, "DCS", "DCS-SimpleRadio-Standalone", dcsPath, dcsFound, batchMode) || needsSave

	if needsSave {
		if err := common.SaveConfig(config); err != nil {
			return fmt.Errorf("unable to save config: %w", err)
		}
	}

	return nil
}
