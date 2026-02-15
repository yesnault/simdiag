package common

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// SelectTemplateInteractive allows the user to choose a template interactively
func SelectTemplateInteractive(templatesDir string, deviceName string) (string, error) {
	// Load all available templates
	templates, err := FindTemplates(templatesDir)
	if err != nil {
		return "", fmt.Errorf("error loading templates: %w", err)
	}

	if len(templates) == 0 {
		return "", fmt.Errorf("no template found in %s", templatesDir)
	}

	// Sort templates by name
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})

	// Display options
	fmt.Printf("\n=== Template for: %s ===\n\n", deviceName)

	for i, template := range templates {
		fmt.Printf("%3d. %-40s [B:%d A:%d H:%d]\n",
			i+1,
			template.Name,
			len(template.Buttons),
			len(template.Axes),
			len(template.Hats))
	}
	fmt.Printf("%3d. Ignore this device\n", len(templates)+1)

	// Read choice
	fmt.Print("\nChoose a template (number): ")
	input, err := ReadLine()
	if err != nil {
		return "", err
	}
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(templates)+1 {
		return "", fmt.Errorf("invalid choice")
	}

	// "Ignore" option
	if choice == len(templates)+1 {
		return "", nil
	}

	return templates[choice-1].FilePath, nil
}

// GetAllDevicesFromProfiles extracts all unique devices from all profiles
func GetAllDevicesFromProfiles(profiles *ProfileCollection) []*Device {
	deviceMap := make(map[string]*Device)

	for _, profile := range profiles.Profiles {
		for guid, device := range profile.Devices {
			if _, exists := deviceMap[guid]; !exists {
				deviceMap[guid] = device
			}
		}
	}

	// Mark virtual devices (vJoy, Thrustmaster Combined, etc.)
	MarkVirtualDevicesInMap(deviceMap)

	// Convert to slice
	var devices []*Device
	for _, device := range deviceMap {
		devices = append(devices, device)
	}

	// Sort by name for consistent display, with virtual devices at the end
	sort.Slice(devices, func(i, j int) bool {
		// Virtual devices go to the end
		if devices[i].IsVirtual != devices[j].IsVirtual {
			return !devices[i].IsVirtual // Non-virtual first
		}
		return devices[i].Name < devices[j].Name
	})

	return devices
}

// AskTemplatesDirectory asks for the templates directory
func AskTemplatesDirectory(config *Config) string {
	defaultDir := getTemplatesDefaultDirectory(config)

	for {
		templatesDir := promptForTemplatesDirectory(defaultDir)
		if templatesDir == "" {
			continue
		}

		if err := validateTemplatesDirectory(templatesDir); err != nil {
			fmt.Println(err)
			continue
		}

		return templatesDir
	}
}

// getTemplatesDefaultDirectory gets the default templates directory
func getTemplatesDefaultDirectory(config *Config) string {
	defaultDir := config.GetCommonTemplatesDirectory()
	if defaultDir == "" {
		defaultDir = "./templates"
	}
	return defaultDir
}

// promptForTemplatesDirectory prompts for templates directory input
func promptForTemplatesDirectory(defaultDir string) string {
	fmt.Printf("\n=== Templates directory ===\n")
	fmt.Printf("Enter the path to the folder containing SVG templates\n")
	fmt.Printf("(leave empty to use: %s): ", defaultDir)

	templatesDir, err := ReadLine()
	if err != nil {
		fmt.Println("⚠ Error reading input")
		return ""
	}

	if templatesDir == "" {
		templatesDir = defaultDir
		fmt.Printf("Using default: %s\n", templatesDir)
	}

	return templatesDir
}

// validateTemplatesDirectory validates that a templates directory exists and contains SVG files
func validateTemplatesDirectory(templatesDir string) error {
	if templatesDir == "" {
		return fmt.Errorf("⚠ Templates directory is required")
	}

	if _, statErr := os.Stat(templatesDir); os.IsNotExist(statErr) {
		return fmt.Errorf("❌ Directory does not exist: %s", templatesDir)
	}

	hasSVG, err := containsSVGFiles(templatesDir)
	if err != nil {
		return fmt.Errorf("⚠ Error checking directory: %v", err)
	}

	if !hasSVG {
		return fmt.Errorf("❌ No SVG files found in: %s\nPlease provide a directory containing SVG template files", templatesDir)
	}

	return nil
}

// containsSVGFiles checks if a directory contains at least one .svg file
func containsSVGFiles(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".svg") {
			return true, nil
		}
	}

	return false, nil
}

// SelectMultipleModules allows user to select multiple modules from a list
// Returns normalized module names for config keys
func SelectMultipleModules(availableModules []string, excludeModule string) []string {
	if len(availableModules) == 0 {
		return nil
	}

	// Filter out the excluded module (compare normalized names)
	normalizedExclude := NormalizeModuleName(excludeModule)
	filteredModules := make([]string, 0)
	for _, mod := range availableModules {
		if NormalizeModuleName(mod) != normalizedExclude {
			filteredModules = append(filteredModules, mod)
		}
	}

	if len(filteredModules) == 0 {
		return nil
	}

	fmt.Println("\n=== Select modules to apply configuration ===")
	fmt.Println("Enter module numbers separated by commas (e.g., 1,3,5)")
	fmt.Println("Or enter 'all' to select all modules")
	fmt.Println()

	for i, module := range filteredModules {
		fmt.Printf("%d. %s\n", i+1, module)
	}

	fmt.Print("\nYour selection: ")
	input, err := ReadLine()
	if err != nil {
		return nil
	}

	input = strings.ToLower(input)

	// Collect selected modules (original names for display, but will normalize for return)
	var selectedOriginalNames []string

	// Handle "all" selection
	if input == "all" {
		selectedOriginalNames = filteredModules
	} else {
		// Parse comma-separated numbers
		parts := strings.Split(input, ",")

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			choice, err := strconv.Atoi(part)
			if err != nil || choice < 1 || choice > len(filteredModules) {
				fmt.Printf("⚠ Invalid choice: %s\n", part)
				continue
			}

			selectedOriginalNames = append(selectedOriginalNames, filteredModules[choice-1])
		}
	}

	if len(selectedOriginalNames) == 0 {
		return nil
	}

	// Normalize all selected module names for config keys
	normalizedModules := make([]string, len(selectedOriginalNames))
	for i, mod := range selectedOriginalNames {
		normalizedModules[i] = NormalizeModuleName(mod)
	}

	// Display what was selected (using original names for clarity)
	fmt.Printf("\n✓ Selected: %s\n", strings.Join(selectedOriginalNames, ", "))

	return normalizedModules
}

// AskGremlinsProfilePath asks for the Gremlins profile file path (optional)
func AskGremlinsProfilePath(config *Config) string {
	// Get common default value from existing configurations
	defaultPath := config.GetCommonGremlinsProfilePath()

	fmt.Printf("\n=== Gremlins Profile Configuration (Optional) ===\n")
	fmt.Println("Gremlins (Joystick Gremlin) allows macro and script bindings integration.")

	// Ask if user wants to configure Gremlins
	fmt.Print("\nDo you want to configure Gremlins profile? (Y/n): ")
	wantGremlin, err := ReadLine()
	if err == nil {
		wantGremlin = strings.ToLower(wantGremlin)
		if wantGremlin == "n" || wantGremlin == "no" {
			fmt.Println("⊘ Gremlins configuration skipped")
			return ""
		}
	}

	for {
		if defaultPath != "" {
			fmt.Printf("\nEnter the full path to your Gremlins profile file (.xml)\n")
			fmt.Printf("(leave empty to use: %s): ", defaultPath)
		} else {
			fmt.Printf("\nEnter the full path to your Gremlins profile file (.xml)\n")
			fmt.Printf("(or leave empty to skip): ")
		}

		profilePath, err := ReadLine()
		if err != nil {
			fmt.Println("⚠ Error reading input")
			continue
		}

		// Use default if empty and default exists
		if profilePath == "" && defaultPath != "" {
			profilePath = defaultPath
			fmt.Printf("Using default: %s\n", profilePath)
		}

		// Allow skipping if empty and no default
		if profilePath == "" {
			fmt.Println("⊘ Gremlins configuration skipped")
			return ""
		}

		// Check that the file exists
		if _, err := os.Stat(profilePath); os.IsNotExist(err) {
			fmt.Printf("❌ File does not exist: %s\n", profilePath)
			fmt.Print("Try again? (Y/n): ")
			retry, _ := ReadLine()
			retry = strings.ToLower(retry)
			if retry == "n" || retry == "no" {
				fmt.Println("⊘ Gremlins configuration skipped")
				return ""
			}
			continue
		}

		// Check that it's an XML file
		if !strings.HasSuffix(strings.ToLower(profilePath), ".xml") {
			fmt.Printf("❌ Gremlins profile must be an XML file (.xml): %s\n", profilePath)
			fmt.Print("Try again? (Y/n): ")
			retry, _ := ReadLine()
			retry = strings.ToLower(retry)
			if retry == "n" || retry == "no" {
				fmt.Println("⊘ Gremlins configuration skipped")
				return ""
			}
			continue
		}

		return profilePath
	}
}

// AskTargetProfilePath asks for the TARGET profile file path (optional)
// Returns the profile path and device mappings
func AskTargetProfilePath(config *Config, devices []*Device) (string, []TargetDeviceMapping) {
	// Ensure external functions are available
	if ExtFuncs == nil {
		return "", nil
	}
	// Get common default value from existing configurations
	defaultPath := config.GetCommonTargetProfilePath()

	fmt.Printf("\n=== Thrustmaster TARGET Configuration (Optional) ===\n")
	fmt.Println("TARGET allows Thrustmaster device remapping and macro integration.")

	// Ask if user wants to configure TARGET
	fmt.Print("\nDo you want to configure a TARGET profile? (Y/n): ")
	wantTarget, err := ReadLine()
	if err == nil {
		wantTarget = strings.ToLower(wantTarget)
		if wantTarget == "n" || wantTarget == "no" {
			fmt.Println("⊘ TARGET configuration skipped")
			return "", nil
		}
	}

	var profilePath string

	for {
		if defaultPath != "" {
			fmt.Printf("\nEnter the full path to your TARGET profile file (.fcf)\n")
			fmt.Printf("(leave empty to use: %s): ", defaultPath)
		} else {
			fmt.Printf("\nEnter the full path to your TARGET profile file (.fcf)\n")
			fmt.Printf("(or leave empty to skip): ")
		}

		profilePath, err = ReadLine()
		if err != nil {
			fmt.Println("⚠ Error reading input")
			continue
		}

		// Use default if empty and default exists
		if profilePath == "" && defaultPath != "" {
			profilePath = defaultPath
			fmt.Printf("Using default: %s\n", profilePath)
		}

		// Allow skipping if empty and no default
		if profilePath == "" {
			fmt.Println("⊘ TARGET configuration skipped")
			return "", nil
		}

		// Check that the file exists
		if _, statErr := os.Stat(profilePath); os.IsNotExist(statErr) {
			fmt.Printf("❌ File does not exist: %s\n", profilePath)
			fmt.Print("Try again? (Y/n): ")
			retry, _ := ReadLine()
			retry = strings.ToLower(retry)
			if retry == "n" || retry == "no" {
				fmt.Println("⊘ TARGET configuration skipped")
				return "", nil
			}
			continue
		}

		// Check that it's an FCF file
		if !strings.HasSuffix(strings.ToLower(profilePath), ".fcf") {
			fmt.Printf("❌ TARGET profile must be an FCF file (.fcf): %s\n", profilePath)
			fmt.Print("Try again? (Y/n): ")
			retry, _ := ReadLine()
			retry = strings.ToLower(retry)
			if retry == "n" || retry == "no" {
				fmt.Println("⊘ TARGET configuration skipped")
				return "", nil
			}
			continue
		}

		break
	}

	// Get the TARGET device numbers from the profile
	targetDeviceNumbers, err := ExtFuncs.GetTargetDeviceNumbers(profilePath)
	if err != nil {
		fmt.Printf("⚠ Error reading TARGET profile: %v\n", err)
		return profilePath, nil
	}

	if len(targetDeviceNumbers) == 0 {
		fmt.Println("⚠ No devices found in TARGET profile")
		return profilePath, nil
	}

	// Filter out virtual devices - only show physical devices for TARGET mapping
	physicalDevices := FilterPhysicalDevices(devices)

	if len(physicalDevices) == 0 {
		fmt.Println("⚠ No physical devices found for TARGET mapping")
		return profilePath, nil
	}

	// Try auto-detection first
	fmt.Println("\n=== TARGET Device Mapping ===")
	autoMappings := ExtFuncs.AutoMatchTargetDevices(targetDeviceNumbers, physicalDevices)

	// Show auto-detected mappings
	if len(autoMappings) > 0 {
		fmt.Println("Auto-detected mappings:")
		for _, m := range autoMappings {
			fmt.Printf("  ✓ TARGET %d (%s) → %s\n", m.DeviceNumber, ExtFuncs.TargetDeviceNumberToName(m.DeviceNumber), m.DeviceName)
		}
	}

	// Check for unmatched devices
	unmatchedDevices := ExtFuncs.GetUnmatchedTargetDevices(targetDeviceNumbers, autoMappings)

	deviceMappings := make([]TargetDeviceMapping, 0)
	deviceMappings = append(deviceMappings, autoMappings...)

	// If all devices were auto-matched, ask for confirmation
	if len(unmatchedDevices) == 0 {
		fmt.Print("\nUse these mappings? (Y/n): ")
		confirm, _ := ReadLine()
		confirm = strings.ToLower(confirm)
		if confirm == "n" || confirm == "no" {
			// User wants to configure manually - clear auto mappings
			deviceMappings = make([]TargetDeviceMapping, 0)
			unmatchedDevices = targetDeviceNumbers
		}
	}

	// Ask user to map remaining unmatched devices
	if len(unmatchedDevices) > 0 {
		if len(autoMappings) > 0 {
			fmt.Println("\nManual mapping required for remaining devices:")
		} else {
			fmt.Println("Map TARGET device numbers to your physical devices:")
		}
		fmt.Println()

		// Build list of available devices (exclude already matched ones)
		usedGUIDs := make(map[string]bool)
		for _, m := range deviceMappings {
			usedGUIDs[m.DeviceGUID] = true
		}

		availableDevices := make([]*Device, 0)
		for _, device := range physicalDevices {
			if !usedGUIDs[device.GUID] {
				availableDevices = append(availableDevices, device)
			}
		}

		for _, targetNum := range unmatchedDevices {
			targetName := ExtFuncs.TargetDeviceNumberToName(targetNum)
			fmt.Printf("TARGET Device %d (%s):\n", targetNum, targetName)

			// Display available physical devices
			for i, device := range availableDevices {
				fmt.Printf("  %d. %s\n", i+1, device.Name)
			}
			fmt.Printf("  %d. Skip this device\n", len(availableDevices)+1)

			fmt.Print("Choose (number): ")
			input, err := ReadLine()
			if err != nil {
				continue
			}

			choice, err := strconv.Atoi(input)
			if err != nil || choice < 1 || choice > len(availableDevices)+1 {
				fmt.Println("⚠ Invalid choice, skipping this device")
				continue
			}

			// Skip option
			if choice == len(availableDevices)+1 {
				fmt.Printf("⊘ Skipping TARGET device %d\n", targetNum)
				continue
			}

			selectedDevice := availableDevices[choice-1]
			mapping := TargetDeviceMapping{
				DeviceNumber: targetNum,
				DeviceGUID:   selectedDevice.GUID,
				DeviceName:   selectedDevice.Name,
			}
			deviceMappings = append(deviceMappings, mapping)
			fmt.Printf("✓ TARGET %d → %s\n", targetNum, selectedDevice.Name)

			// Remove from available devices
			newAvailable := make([]*Device, 0)
			for _, d := range availableDevices {
				if d.GUID != selectedDevice.GUID {
					newAvailable = append(newAvailable, d)
				}
			}
			availableDevices = newAvailable
		}
	}

	if len(deviceMappings) == 0 {
		fmt.Println("⚠ No TARGET device mappings configured")
	}

	return profilePath, deviceMappings
}
