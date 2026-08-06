package common

import (
	"cmp"
	"fmt"
	"os"
	"slices"
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
	slices.SortFunc(templates, func(a, b *Template) int {
		return cmp.Compare(a.Name, b.Name)
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
	slices.SortFunc(devices, func(a, b *Device) int {
		if a.IsVirtual != b.IsVirtual {
			if a.IsVirtual {
				return 1 // Non-virtual first
			}
			return -1
		}
		return cmp.Compare(a.Name, b.Name)
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
	defaultDir := config.TemplatesDirectory
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
		return fmt.Errorf("⚠ Error checking directory: %w", err)
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

	fmt.Printf("\n=== Thrustmaster TARGET Configuration (Optional) ===\n")
	fmt.Println("TARGET allows Thrustmaster device remapping and macro integration.")

	if !askYesNoDefaultYes("\nDo you want to configure a TARGET profile?") {
		fmt.Println("⊘ TARGET configuration skipped")
		return "", nil
	}

	profilePath := promptTargetProfilePath(config.GetCommonTargetProfilePath())
	if profilePath == "" {
		return "", nil
	}

	return profilePath, mapTargetDevices(profilePath, devices)
}

// askYesNoDefaultYes prompts for confirmation, treating an empty answer and any
// read error as "yes".
func askYesNoDefaultYes(question string) bool {
	fmt.Printf("%s (Y/n): ", question)
	answer, err := ReadLine()
	if err != nil {
		return true
	}
	answer = strings.ToLower(answer)
	return answer != "n" && answer != "no"
}

// promptTargetProfilePath asks for an existing .fcf profile, re-prompting until a
// valid one is given. Returns "" when the user gives up or skips.
func promptTargetProfilePath(defaultPath string) string {
	for {
		fmt.Printf("\nEnter the full path to your TARGET profile file (.fcf)\n")
		if defaultPath != "" {
			fmt.Printf("(leave empty to use: %s): ", defaultPath)
		} else {
			fmt.Printf("(or leave empty to skip): ")
		}

		profilePath, err := ReadLine()
		if err != nil {
			fmt.Println("⚠ Error reading input")
			continue
		}

		if profilePath == "" && defaultPath != "" {
			profilePath = defaultPath
			fmt.Printf("Using default: %s\n", profilePath)
		}

		// Empty with no default means "skip"
		if profilePath == "" {
			fmt.Println("⊘ TARGET configuration skipped")
			return ""
		}

		problem := ""
		switch {
		case !fileExists(profilePath):
			problem = fmt.Sprintf("❌ File does not exist: %s", profilePath)
		case !strings.HasSuffix(strings.ToLower(profilePath), ".fcf"):
			problem = fmt.Sprintf("❌ TARGET profile must be an FCF file (.fcf): %s", profilePath)
		}

		if problem == "" {
			return profilePath
		}

		fmt.Println(problem)
		if !askYesNoDefaultYes("Try again?") {
			fmt.Println("⊘ TARGET configuration skipped")
			return ""
		}
	}
}

// fileExists reports whether path names an existing file.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// mapTargetDevices associates each TARGET device number in the profile with one of
// the user's physical devices, auto-detecting what it can and asking for the rest.
func mapTargetDevices(profilePath string, devices []*Device) []TargetDeviceMapping {
	targetDeviceNumbers, err := ExtFuncs.GetTargetDeviceNumbers(profilePath)
	if err != nil {
		fmt.Printf("⚠ Error reading TARGET profile: %v\n", err)
		return nil
	}
	if len(targetDeviceNumbers) == 0 {
		fmt.Println("⚠ No devices found in TARGET profile")
		return nil
	}

	// Only physical devices can back a TARGET device
	physicalDevices := FilterPhysicalDevices(devices)
	if len(physicalDevices) == 0 {
		fmt.Println("⚠ No physical devices found for TARGET mapping")
		return nil
	}

	fmt.Println("\n=== TARGET Device Mapping ===")
	autoMappings := ExtFuncs.AutoMatchTargetDevices(targetDeviceNumbers, physicalDevices)

	if len(autoMappings) > 0 {
		fmt.Println("Auto-detected mappings:")
		for _, m := range autoMappings {
			fmt.Printf("  ✓ TARGET %d (%s) → %s\n", m.DeviceNumber, ExtFuncs.TargetDeviceNumberToName(m.DeviceNumber), m.DeviceName)
		}
	}

	unmatchedDevices := ExtFuncs.GetUnmatchedTargetDevices(targetDeviceNumbers, autoMappings)

	deviceMappings := make([]TargetDeviceMapping, 0, len(targetDeviceNumbers))
	deviceMappings = append(deviceMappings, autoMappings...)

	// Everything matched automatically: confirm, or fall back to full manual mapping
	if len(unmatchedDevices) == 0 && !askYesNoDefaultYes("\nUse these mappings?") {
		deviceMappings = deviceMappings[:0]
		unmatchedDevices = targetDeviceNumbers
	}

	if len(unmatchedDevices) > 0 {
		if len(autoMappings) > 0 {
			fmt.Println("\nManual mapping required for remaining devices:")
		} else {
			fmt.Println("Map TARGET device numbers to your physical devices:")
		}
		fmt.Println()

		deviceMappings = askManualTargetMappings(unmatchedDevices, physicalDevices, deviceMappings)
	}

	if len(deviceMappings) == 0 {
		fmt.Println("⚠ No TARGET device mappings configured")
	}

	return deviceMappings
}

// askManualTargetMappings walks the user through the TARGET devices that could not
// be auto-detected, offering the physical devices not yet spoken for.
func askManualTargetMappings(unmatchedDevices []int, physicalDevices []*Device, deviceMappings []TargetDeviceMapping) []TargetDeviceMapping {
	// Only offer devices that are not already mapped
	usedGUIDs := make(map[string]bool, len(deviceMappings))
	for _, m := range deviceMappings {
		usedGUIDs[m.DeviceGUID] = true
	}

	availableDevices := make([]*Device, 0, len(physicalDevices))
	for _, device := range physicalDevices {
		if !usedGUIDs[device.GUID] {
			availableDevices = append(availableDevices, device)
		}
	}

	for _, targetNum := range unmatchedDevices {
		fmt.Printf("TARGET Device %d (%s):\n", targetNum, ExtFuncs.TargetDeviceNumberToName(targetNum))

		for i, device := range availableDevices {
			fmt.Printf("  %d. %s\n", i+1, device.Name)
		}
		skipChoice := len(availableDevices) + 1
		fmt.Printf("  %d. Skip this device\n", skipChoice)

		fmt.Print("Choose (number): ")
		input, err := ReadLine()
		if err != nil {
			continue
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > skipChoice {
			fmt.Println("⚠ Invalid choice, skipping this device")
			continue
		}

		if choice == skipChoice {
			fmt.Printf("⊘ Skipping TARGET device %d\n", targetNum)
			continue
		}

		selectedDevice := availableDevices[choice-1]
		deviceMappings = append(deviceMappings, TargetDeviceMapping{
			DeviceNumber: targetNum,
			DeviceGUID:   selectedDevice.GUID,
			DeviceName:   selectedDevice.Name,
		})
		fmt.Printf("✓ TARGET %d → %s\n", targetNum, selectedDevice.Name)

		// A physical device backs at most one TARGET device
		availableDevices = slices.Delete(availableDevices, choice-1, choice)
	}

	return deviceMappings
}
