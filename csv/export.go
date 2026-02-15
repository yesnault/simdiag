package csv

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"simdiag/common"
)

// ExportToCSV exports all bindings to a CSV file
func ExportToCSV(exportDevices []*common.ExportDevice, outputPath string, config *common.Config) error {
	if len(exportDevices) == 0 {
		return nil
	}

	// Collect all rows, deduplicating by action + virtual device/input
	// Key: "action|virtualDevice|virtualInput" -> row data
	// Keep the row with physical device info if available
	rowMap := make(map[string]*csvRowData)

	// Track physical device mappings by virtual device+input
	// Key: "normalizedVirtualDevice|virtualInput" -> {physicalDevice, physicalInput, physicalDeviceGUID, modifier}
	type physicalMapping struct {
		device     string
		input      string
		deviceGUID string
		modifier   string
	}
	physicalMappings := make(map[string]*physicalMapping)

	// Process each export device
	for _, exportDevice := range exportDevices {
		// Get template path (relative to templates_directory)
		templatePath := ""
		if exportDevice.Template != nil && exportDevice.Template.FilePath != "" {
			if config != nil && config.TemplatesDirectory != "" {
				templatePath = common.MakeRelativePath(exportDevice.Template.FilePath, config.TemplatesDirectory)
			} else {
				templatePath = exportDevice.Template.FilePath
			}
		}
		// Determine simulator name
		simulator := exportDevice.Profile.SimType.GetConfigKey()

		// Determine module name
		module := ""
		if exportDevice.Profile.SimType == common.DCSWorld && exportDevice.Profile.Module != "" {
			module = exportDevice.Profile.Module
		} else if exportDevice.Profile.SimType == common.IL2Sturmovik {
			module = "il2"
		}

		// Process each binding
		for _, binding := range exportDevice.Profile.Bindings {
			// Note: We now include Switch/Modifier bindings in CSV so they can be regenerated
			// These are important for displaying mode switching information in diagrams

			// Use the simulator from the profile - SRS and OpenKneeboard are tools, not simulators
			// They are used within DCS or IL2, so keep the original simulator value
			bindingSimulator := simulator

			// Get action text (what's displayed to the user)
			action := getBindingDisplayText(binding)

			// Clean up action text for CSV
			action = strings.ReplaceAll(action, "\"", "\"\"") // Escape quotes
			action = strings.ReplaceAll(action, "\n", " ")    // Remove line breaks
			action = strings.TrimSpace(action)

			// Get modifier text, modifier device, and modifier number
			modifierText := ""
			modifierDeviceName := ""
			modifierNum := ""

			// Get modifier number from binding (already assigned by AssignModifierNumbers)
			if binding.ModifierNum > 0 {
				modifierNum = fmt.Sprintf("%d", binding.ModifierNum)
			}

			if len(binding.Modifiers) > 0 {
				modifierKeys := make([]string, 0)
				for _, mod := range binding.Modifiers {
					if len(mod.Keys) > 0 {
						// Extract just the button name from modifier key
						// e.g., "JOY_BTN105" -> "BTN105", "GREMLINS_MODE_Shift" -> "Shift"
						// Also get device name from the first modifier key
						for _, key := range mod.Keys {
							// Get modifier device name from ModifierDeviceMap
							if modifierDeviceName == "" && strings.HasPrefix(key, "JOY_BTN") {
								if modInfo, exists := exportDevice.Profile.ModifierDeviceMap[key]; exists {
									modifierDeviceName = modInfo.DeviceName
								}
							}

							switch {
							case strings.HasPrefix(key, "JOY_BTN"):
								modifierKeys = append(modifierKeys, strings.TrimPrefix(key, "JOY_"))
							case strings.HasPrefix(key, "GREMLINS_MODE_"):
								modifierKeys = append(modifierKeys, strings.TrimPrefix(key, "GREMLINS_MODE_"))
							default:
								modifierKeys = append(modifierKeys, key)
							}
						}
					}
				}
				modifierText = strings.Join(modifierKeys, " + ")
			}

			// Get input text
			inputText := ""
			switch binding.InputType {
			case common.Button:
				inputText = "BTN" + binding.InputID
			case common.Axis:
				inputText = "Axis " + binding.InputID
			case common.Hat:
				inputText = "POV " + binding.InputID
			}

			// Calculate template key that will be replaced in the SVG template
			// Strip _OFF suffix so BTN25_OFF maps to Button_25 (not Button_25_OFF)
			templateInputID := strings.TrimSuffix(binding.InputID, "_OFF")

			var templateKey string
			switch binding.InputType {
			case common.Button:
				templateKey = fmt.Sprintf("Button_%s", templateInputID)
			case common.Axis:
				templateKey = fmt.Sprintf("AXIS_%s", strings.ToUpper(templateInputID))
			case common.Hat:
				templateKey = fmt.Sprintf("POV_%s", strings.ToUpper(templateInputID))
			}

			// Skip keyboard bindings - we only want joystick/controller bindings
			isKeyboard := strings.EqualFold(binding.DeviceName, "Keyboard")

			if isKeyboard {
				continue
			}

			// Determine if device is virtual (vJoy, etc.) or physical
			// vJoy devices should go in Virtual Device/Input columns
			physicalDevice := ""
			physicalInput := ""
			physicalDeviceGUID := ""
			virtualDevice := binding.VirtualDevice
			virtualInput := binding.VirtualInput

			// Normalize key order in virtualInput for consistent comparison
			// This ensures "F + LShift" and "LShift + F" are treated as the same
			if virtualDevice == "Keyboard" && virtualInput != "" {
				// Split by " + ", normalize order, rejoin
				keys := strings.Split(virtualInput, " + ")
				normalizedKeys := common.NormalizeKeyOrder(keys)
				virtualInput = strings.Join(normalizedKeys, " + ")
			}

			isVirtualDevice := strings.Contains(strings.ToLower(binding.DeviceName), "vjoy")

			if isVirtualDevice {
				// Device is virtual (vJoy) - put in virtual columns
				// Normalize name to "vJoy Device <GUID[:8]>" for clarity
				shortGUID := common.NormalizeGUIDShort(binding.DeviceGUID)
				if shortGUID != "" {
					virtualDevice = fmt.Sprintf("vJoy Device %s", shortGUID)
				} else {
					virtualDevice = binding.DeviceName
				}
				virtualInput = inputText
			} else {
				// Device is physical - put in physical columns
				physicalDevice = binding.DeviceName
				physicalInput = inputText
				physicalDeviceGUID = binding.DeviceGUID
			}

			// Normalize vJoy device name for deduplication
			// Use first 8 chars of GUID or extracted number to differentiate vJoy devices
			normalizedVirtualDevice := virtualDevice
			if strings.Contains(strings.ToLower(virtualDevice), "vjoy") {
				// Extract identifier from the virtual device name
				// Format: "vJoy Device abc12345" or "vJoy Device #1"
				normalizedVirtualDevice = normalizeVJoyName(virtualDevice)
			}

			// Track physical mappings by virtual device+input
			// This allows us to propagate physical info to all actions on the same virtual button
			if physicalDevice != "" && virtualInput != "" {
				physicalKey := fmt.Sprintf("%s|%s", normalizedVirtualDevice, virtualInput)
				if _, exists := physicalMappings[physicalKey]; !exists {
					physicalMappings[physicalKey] = &physicalMapping{
						device:     physicalDevice,
						input:      physicalInput,
						deviceGUID: physicalDeviceGUID,
						modifier:   modifierText,
					}
				}
			}

			// Create deduplication key: action + virtual device + virtual input + physical device + physical input + module
			// We include physical device/input to avoid deduplicating different physical buttons
			// that happen to send the same virtual keys (e.g., Joystick BTN5 and Throttle BTN12 both sending LShift+G)
			// We include module to avoid deduplicating SRS/OpenKneeboard bindings across different modules
			// Note: modifier is NOT included - same physical button with different modifiers are still deduplicated
			dedupKey := fmt.Sprintf("%s|%s|%s|%s|%s|%s", action, normalizedVirtualDevice, virtualInput, physicalDevice, physicalInput, module)

			row := &csvRowData{
				simulator:          bindingSimulator,
				module:             module,
				action:             action,
				modifier:           modifierText,
				modifierDevice:     modifierDeviceName,
				modifierNum:        modifierNum,
				physicalDevice:     physicalDevice,
				physicalInput:      physicalInput,
				physicalDeviceGUID: physicalDeviceGUID,
				virtualDevice:      virtualDevice,
				virtualInput:       virtualInput,
				templateKey:        templateKey,
				templatePath:       templatePath,
			}

			// Check if we already have this entry
			if existing, exists := rowMap[dedupKey]; exists {
				// Keep the one with physical device info (it also has the correct modifier from Gremlins)
				if existing.physicalDevice == "" && physicalDevice != "" {
					rowMap[dedupKey] = row
				}
				// If existing has no ACTION text (just "TARGET"), prefer the new one with actual description
				if strings.HasPrefix(existing.action, "TARGET") && !strings.HasPrefix(action, "TARGET") {
					rowMap[dedupKey] = row
				}
				// Otherwise keep existing (which already has physical info or both are empty)
			} else {
				rowMap[dedupKey] = row
			}
		}
	}

	// Propagate physical device info to all rows sharing the same virtual button
	// This ensures that if Gremlins maps Throttle BTN8 -> vJoy2 BTN8,
	// ALL actions on vJoy2 BTN8 get the physical device info
	for _, row := range rowMap {
		if row.physicalDevice == "" && row.virtualInput != "" {
			normalizedVD := row.virtualDevice
			if strings.Contains(strings.ToLower(row.virtualDevice), "vjoy") {
				normalizedVD = normalizeVJoyName(row.virtualDevice)
			}
			physicalKey := fmt.Sprintf("%s|%s", normalizedVD, row.virtualInput)
			if mapping, exists := physicalMappings[physicalKey]; exists {
				row.physicalDevice = mapping.device
				row.physicalInput = mapping.input
				row.physicalDeviceGUID = mapping.deviceGUID
				// Also propagate the modifier from Gremlins if the row doesn't have one
				if row.modifier == "" && mapping.modifier != "" {
					row.modifier = mapping.modifier
				}
			}
		}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Create CSV file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating CSV file: %w", err)
	}
	defer file.Close()

	// Write CSV header
	if _, err := file.WriteString(GetHeaderString()); err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Write all unique rows
	for _, row := range rowMap {
		csvRow := fmt.Sprintf("%s,%s,\"%s\",%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			escapeCsvField(row.simulator),
			escapeCsvField(row.module),
			row.action, // Already escaped above
			escapeCsvField(row.modifier),
			escapeCsvField(row.modifierDevice),
			escapeCsvField(row.modifierNum),
			escapeCsvField(row.physicalDevice),
			escapeCsvField(row.physicalInput),
			escapeCsvField(row.physicalDeviceGUID),
			escapeCsvField(row.virtualDevice),
			escapeCsvField(row.virtualInput),
			escapeCsvField(row.templateKey),
			escapeCsvField(row.templatePath),
		)

		if _, err := file.WriteString(csvRow); err != nil {
			return fmt.Errorf("error writing CSV row: %w", err)
		}
	}

	return nil
}

// getBindingDisplayText returns the best display text for a binding
// For IL-2: use Description (e.g. "Suralimentation") if available
// For DCS/SRS: use Action
func getBindingDisplayText(binding common.Binding) string {
	if binding.Description != "" {
		return binding.Description
	}
	return binding.Action
}

// escapeCsvField escapes a field for CSV format
func escapeCsvField(field string) string {
	// If field contains comma, quote, or newline, wrap in quotes
	if strings.Contains(field, ",") || strings.Contains(field, "\"") || strings.Contains(field, "\n") {
		// Escape quotes by doubling them
		field = strings.ReplaceAll(field, "\"", "\"\"")
		return "\"" + field + "\""
	}
	return field
}

// normalizeVJoyName extracts a consistent identifier from vJoy device names
// "vJoy Device" -> "vJoy"
// "vJoy Device #1" -> "vJoy_1"
// "vJoy Device #2" -> "vJoy_2"
// "vJoy Device abc12345" -> "vJoy_abc12345"
func normalizeVJoyName(name string) string {
	// Try to extract GUID prefix from "vJoy Device abc12345"
	reGUID := regexp.MustCompile(`(?i)vjoy\s+device\s+([a-f0-9]{8})`)
	matchesGUID := reGUID.FindStringSubmatch(name)
	if len(matchesGUID) >= 2 {
		return fmt.Sprintf("vJoy_%s", strings.ToLower(matchesGUID[1]))
	}

	// Try to extract number from "vJoy Device #X"
	reNum := regexp.MustCompile(`#(\d+)`)
	matchesNum := reNum.FindStringSubmatch(name)
	if len(matchesNum) >= 2 {
		return fmt.Sprintf("vJoy_%s", matchesNum[1])
	}

	return "vJoy"
}
