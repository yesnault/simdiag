package openkneeboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"simdiag/common"
)

// Configurator handles OpenKneeboard configuration
type Configurator struct{}

// NewConfigurator creates a new OpenKneeboard configurator
func NewConfigurator() *Configurator {
	return &Configurator{}
}

// GetName returns the name of the configurator
func (c *Configurator) GetName() string {
	return "OpenKneeboard"
}

// Configure prompts the user to configure OpenKneeboard Profiles.json path
func (c *Configurator) Configure(config *common.Config, batchMode bool) error {
	if config == nil {
		config = &common.Config{Simulators: make(map[string]*common.SimulatorConfig)}
	}

	// In batch mode, skip configuration
	if batchMode {
		return nil
	}

	// Check if OpenKneeboard path is already configured
	if config.OpenKneeboardProfilesFilepath != "" {
		return nil
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("OpenKneeboard Configuration (Optional)")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("\nOpenKneeboard allows kneeboard control bindings integration.")

	if !common.AskYesNo("Do you want to configure OpenKneeboard?") {
		// User doesn't want to configure OpenKneeboard
		fmt.Println("⊘ OpenKneeboard configuration skipped")
		config.OpenKneeboardProfilesFilepath = ""
		if err := common.SaveConfig(config); err != nil {
			return fmt.Errorf("unable to save config: %w", err)
		}
		return nil
	}

	// Propose default path
	defaultPath := filepath.Join(os.Getenv("LOCALAPPDATA"), "OpenKneeboard", "Settings", "Profiles.json")
	fmt.Printf("\nDefault OpenKneeboard Profiles.json path: %s\n", defaultPath)

	// Check if default path exists
	if _, err := os.Stat(defaultPath); err == nil {
		if common.AskYesNo("Use this default path?") {
			config.OpenKneeboardProfilesFilepath = defaultPath
			if err := common.SaveConfig(config); err != nil {
				return fmt.Errorf("unable to save config: %w", err)
			}
			fmt.Printf("✓ OpenKneeboard Profiles.json path saved: %s\n", defaultPath)
			return nil
		}
	}

	fmt.Print("\nEnter the full path to your OpenKneeboard Profiles.json file: ")
	path, err := common.ReadLine()
	if err != nil || path == "" {
		fmt.Println("⊘ OpenKneeboard configuration skipped")
		return nil
	}
	// Remove quotes if present
	path = strings.Trim(path, "\"'")

	// Verify the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("❌ File not found: %s\n", path)
		return nil
	}

	// Verify it's named Profiles.json
	if filepath.Base(path) != "Profiles.json" {
		fmt.Printf("⚠ Warning: File is not named Profiles.json: %s\n", filepath.Base(path))
		if !common.AskYesNo("Continue anyway?") {
			return nil
		}
	}

	// Path exists, save it
	config.OpenKneeboardProfilesFilepath = path
	if err := common.SaveConfig(config); err != nil {
		return fmt.Errorf("unable to save config: %w", err)
	}
	fmt.Printf("✓ OpenKneeboard Profiles.json path saved: %s\n", path)

	return nil
}
