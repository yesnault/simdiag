package common

import (
	"fmt"
	"os"
	"strings"
)

// ConfigureDrawIOPath prompts user to configure draw.io path for PNG export
func ConfigureDrawIOPath(config *Config) bool {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("draw.io not found for PNG conversion")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("\nTo export PNG files, draw.io Desktop is required.")
	fmt.Println("Download: https://github.com/jgraph/drawio-desktop/releases")
	fmt.Println()

	if !AskYesNo("Do you want to specify the path to draw.io now?") {
		return false
	}

	fmt.Print("\nEnter the full path to draw.io.exe: ")
	path, err := ReadLine()
	if err != nil || path == "" {
		return false
	}
	// Remove quotes if present
	path = strings.Trim(path, "\"'")

	// Verify the path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("❌ File not found: %s\n", path)
		return false
	}

	// Save to config
	if config == nil {
		config = &Config{Simulators: make(map[string]*SimulatorConfig)}
	}
	config.DrawIOPath = path

	if err := SaveConfig(config); err != nil {
		fmt.Printf("⚠ Unable to save config: %v\n", err)
		return false
	}

	fmt.Printf("✓ draw.io path saved: %s\n", path)
	return true
}
