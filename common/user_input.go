package common

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// stdinReader is a shared bufio.Reader for os.Stdin.
// Using a single reader avoids the bug where multiple bufio.NewReader(os.Stdin)
// instances interfere with each other (buffered data gets lost).
var (
	stdinReader     *bufio.Reader
	stdinReaderOnce sync.Once
)

// getStdinReader returns the shared stdin reader (singleton).
func getStdinReader() *bufio.Reader {
	stdinReaderOnce.Do(func() {
		stdinReader = bufio.NewReader(os.Stdin)
	})
	return stdinReader
}

// ReadLine reads a line from stdin, trims whitespace, and returns it.
// Returns the trimmed input and any error.
func ReadLine() (string, error) {
	input, err := getStdinReader().ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// ReadLineWithDefault reads a line from stdin. If the input is empty (or on error),
// returns the provided default value.
func ReadLineWithDefault(defaultValue string) string {
	input, err := ReadLine()
	if err != nil || input == "" {
		return defaultValue
	}
	return input
}

// SelectSimulation prompts user to select a simulator
func SelectSimulation() SimulationType {
	fmt.Println("\n=== Simulation Selection ===")
	fmt.Println("1. DCS World")
	fmt.Println("2. IL-2 Sturmovik")
	fmt.Print("\nChoose (1-2): ")

	input, err := ReadLine()
	if err != nil {
		fmt.Println("Input error, using DCS by default")
		return DCSWorld
	}

	if input == "2" {
		return IL2Sturmovik
	}
	return DCSWorld
}

// GetConfigPath requests configuration file path.
// If config is nil, it will be loaded automatically.
func GetConfigPath(config *Config, simType SimulationType, batchMode bool) string {
	if config == nil {
		var err error
		config, err = LoadConfig()
		if err != nil {
			config = &Config{Simulators: make(map[string]*SimulatorConfig)}
		}
	}

	var defaultPath string
	homeDir, _ := os.UserHomeDir()

	switch simType {
	case DCSWorld:
		dcsConfig := config.GetSimulatorConfig(DCSWorld)
		if dcsConfig != nil && dcsConfig.DCSPath != "" {
			defaultPath = dcsConfig.DCSPath
		} else {
			defaultPath = filepath.Join(homeDir, "Saved Games", "DCS")
		}
	case IL2Sturmovik:
		il2Config := config.GetSimulatorConfig(IL2Sturmovik)
		if il2Config != nil && il2Config.IL2InputPath != "" {
			defaultPath = il2Config.IL2InputPath
		} else {
			defaultPath = filepath.Join("C:\\", "Program Files", "IL-2 Sturmovik Great Battles", "data", "input")
		}
	}

	// In batch mode, always use default path (which may now be a configured custom path)
	if batchMode {
		return defaultPath
	}

	fmt.Printf("\nConfiguration files path (leave empty for: %s): ", defaultPath)
	return ReadLineWithDefault(defaultPath)
}

// SelectModule asks user to select a DCS module to configure
func SelectModule(availableModules []string) string {
	if len(availableModules) == 0 {
		return ""
	}

	if len(availableModules) == 1 {
		return availableModules[0]
	}

	for i, module := range availableModules {
		fmt.Printf("%d. %s\n", i+1, module)
	}
	fmt.Print("\nChoose a module (number): ")

	input, err := ReadLine()
	if err != nil {
		return availableModules[0]
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(availableModules) {
		return availableModules[0]
	}

	return availableModules[choice-1]
}

// AskYesNo prompts user for yes/no confirmation
func AskYesNo(question string) bool {
	for {
		fmt.Printf("%s (Y/n): ", question)
		input, err := ReadLine()
		if err != nil {
			return true // Default: Yes on error
		}

		input = strings.ToLower(input)

		// Accept empty input (default to Yes)
		if input == "" || input == "y" || input == "yes" {
			return true
		}

		// Accept no
		if input == "n" || input == "no" {
			return false
		}

		// Invalid input, ask again
		fmt.Println("⚠ Invalid response. Please answer 'Y' for yes or 'n' for no.")
	}
}

// DisplayProfiles displays found profiles
func DisplayProfiles(collection *ProfileCollection) {
	if collection == nil || len(collection.Profiles) == 0 {
		fmt.Println("\nNo profile found.")
		return
	}

	fmt.Printf("\n=== %d profile(s) found ===\n\n", len(collection.Profiles))

	for i, profile := range collection.Profiles {
		fmt.Printf("--- Profile %d: %s ---\n", i+1, profile.Name)
		fmt.Printf("Type: %s\n", profile.SimType)
		fmt.Printf("Devices: %d\n", len(profile.Devices))
		fmt.Printf("Bindings: %d\n\n", len(profile.Bindings))

		// Display devices
		if len(profile.Devices) > 0 {
			fmt.Println("Devices:")
			for _, device := range profile.Devices {
				fmt.Printf("  - %s (GUID: %s)\n", device.Name, device.GUID)
			}
			fmt.Println()
		}
	}
}

// GetModulesFromProfiles extracts unique module names from profiles (DCS only)
func GetModulesFromProfiles(profiles *ProfileCollection) []string {
	moduleSet := make(map[string]bool)

	for _, profile := range profiles.Profiles {
		if profile.Module != "" {
			moduleSet[profile.Module] = true
		}
	}

	modules := make([]string, 0, len(moduleSet))
	for module := range moduleSet {
		modules = append(modules, module)
	}

	sort.Strings(modules)

	return modules
}
