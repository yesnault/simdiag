package svg

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"simdiag/common"
	simdiagcsv "simdiag/csv"
)

// csvRow represents a single row from the CSV export
type csvRow struct {
	simulator          string
	module             string
	action             string
	modifier           string
	modifierDevice     string // Device name for the modifier button (if different from physical device)
	modifierNum        string // Sequential modifier number from CSV
	physicalDevice     string
	physicalInput      string
	physicalDeviceGUID string // GUID of the physical device (not used for SVG generation, but read for completeness)
	virtualDevice      string
	virtualInput       string
	templateKey        string
	templatePath       string
}

// GenerateSVGFromCSV reads a CSV file and generates SVG/PNG diagrams from it
func GenerateSVGFromCSV(csvPath string, config *common.Config) error {
	// Verify config
	if config == nil {
		return fmt.Errorf("config is required")
	}
	if config.TemplatesDirectory == "" {
		return fmt.Errorf("templates directory not configured")
	}
	if config.OutputDirectory == "" {
		return fmt.Errorf("output directory not configured")
	}

	// Open CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("error opening CSV file: %w", err)
	}
	defer file.Close()

	// Read CSV
	reader := csv.NewReader(file)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("error reading CSV header: %w", err)
	}

	// Validate header and get column indices
	colIndices := make(map[string]int)
	for i, col := range header {
		colIndices[col] = i
	}
	for _, col := range simdiagcsv.AllColumns {
		if _, exists := colIndices[col]; !exists {
			return fmt.Errorf("missing required column: %s", col)
		}
	}

	// Read all rows and group by (Simulator, Module, Template)
	// This matches the batch workflow behavior: all devices using the same template are merged
	// Key: "simulator|module|template" -> list of CSV rows
	groups := make(map[string][]csvRow)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading CSV row: %w", err)
		}

		row := csvRow{
			simulator:          record[colIndices[simdiagcsv.ColSimulator]],
			module:             record[colIndices[simdiagcsv.ColModule]],
			action:             record[colIndices[simdiagcsv.ColAction]],
			modifier:           record[colIndices[simdiagcsv.ColModifier]],
			modifierDevice:     record[colIndices[simdiagcsv.ColModifierDevice]],
			modifierNum:        record[colIndices[simdiagcsv.ColModifierNum]],
			physicalDevice:     record[colIndices[simdiagcsv.ColPhysicalDevice]],
			physicalInput:      record[colIndices[simdiagcsv.ColPhysicalInput]],
			physicalDeviceGUID: record[colIndices[simdiagcsv.ColPhysicalDeviceGUID]],
			virtualDevice:      record[colIndices[simdiagcsv.ColVirtualDevice]],
			virtualInput:       record[colIndices[simdiagcsv.ColVirtualInput]],
			templateKey:        record[colIndices[simdiagcsv.ColTemplateKey]],
			templatePath:       record[colIndices[simdiagcsv.ColTemplate]],
		}

		// Skip rows without physical device (virtual-only bindings)
		if row.physicalDevice == "" {
			continue
		}

		// Skip rows without template
		if row.templatePath == "" {
			continue
		}

		// Create group key - group by template, not by device
		// This allows merging bindings from multiple devices using the same template
		groupKey := fmt.Sprintf("%s|%s|%s", row.simulator, row.module, row.templatePath)
		groups[groupKey] = append(groups[groupKey], row)
	}

	fmt.Printf("\nGenerating diagrams from CSV (%d groups)...\n", len(groups))

	// Process each group
	exportCount := 0
	var allValidationErrors []common.ValidationError
	for groupKey, rows := range groups {
		if len(rows) == 0 {
			continue
		}

		// Parse group key
		parts := strings.Split(groupKey, "|")
		if len(parts) != 3 {
			continue
		}
		simulator := parts[0]
		module := parts[1]
		templatePath := parts[2]

		// Determine simulator type
		simType := parseSimulatorType(simulator)

		// Load template
		absoluteTemplatePath := common.MakeAbsolutePath(templatePath, config.TemplatesDirectory)
		template, err := common.LoadTemplate(absoluteTemplatePath)
		if err != nil {
			fmt.Printf("  ⚠ Error loading template %s: %v\n", templatePath, err)
			continue
		}

		// Collect all unique devices from rows in this group
		// (multiple devices can use the same template)
		deviceMap := make(map[string]*common.Device)
		for _, row := range rows {
			if row.physicalDevice == "" {
				continue
			}

			// Use GUID from CSV (best for round-trip fidelity)
			deviceGUID := row.physicalDeviceGUID
			if deviceGUID == "" {
				// Fallback: search in config
				deviceGUID = findDeviceGUID(row.physicalDevice, config)
			}
			if deviceGUID == "" {
				// Fallback: create a short GUID based on device name hash
				deviceGUID = fmt.Sprintf("%08x-0000-0000-0000-000000000000", hashString(row.physicalDevice))
			}

			// Create device if not already in map
			if _, exists := deviceMap[deviceGUID]; !exists {
				deviceMap[deviceGUID] = &common.Device{
					GUID:      deviceGUID,
					Name:      row.physicalDevice,
					IsVirtual: false,
				}
			}
		}

		// Use first device as representative
		var representativeDevice *common.Device
		var representativeName string
		for guid, device := range deviceMap {
			representativeDevice = device
			representativeName = device.Name
			_ = guid
			break
		}

		if representativeDevice == nil {
			continue
		}

		// Create Profile with merged bindings from all devices
		profile := &common.Profile{
			Name:     representativeName,
			SimType:  simType,
			Module:   module,
			Devices:  deviceMap,
			Bindings: make([]common.Binding, 0),
		}

		// Convert CSV rows to bindings
		for _, row := range rows {
			// Use GUID from CSV (best for round-trip fidelity)
			deviceGUID := row.physicalDeviceGUID
			if deviceGUID == "" {
				// Fallback: search in config
				deviceGUID = findDeviceGUID(row.physicalDevice, config)
			}
			if deviceGUID == "" {
				// Fallback: create a short GUID based on device name hash
				deviceGUID = fmt.Sprintf("%08x-0000-0000-0000-000000000000", hashString(row.physicalDevice))
			}

			binding := csvRowToBinding(row, deviceGUID, row.physicalDevice)
			if binding != nil {
				profile.Bindings = append(profile.Bindings, *binding)
			}
		}

		// Determine output directory
		outputDir := config.OutputDirectory
		switch {
		case simType == common.DCSWorld && module != "":
			normalizedModule := common.NormalizeModuleName(module)
			outputDir = filepath.Join(config.OutputDirectory, "dcs-"+normalizedModule)
		case simType == common.IL2Sturmovik:
			outputDir = filepath.Join(config.OutputDirectory, "il2")
		case simType == common.IL2Korea:
			outputDir = filepath.Join(config.OutputDirectory, "il2-korea")
		}

		// Create title
		title := representativeName
		switch {
		case simType == common.DCSWorld && module != "":
			title = fmt.Sprintf("DCS World / %s", strings.ToUpper(module))
		case simType == common.IL2Sturmovik:
			title = "IL-2 Sturmovik"
		case simType == common.IL2Korea:
			title = "IL-2 Korea"
		}

		// Create ExportDevice
		exportDevice := &common.ExportDevice{
			Device:          representativeDevice,
			Template:        template,
			Profile:         profile,
			OutputDirectory: outputDir,
			SimulatorName:   simType.GetConfigKey(),
			SimdiagVersion:  common.SimdiagVersion,
			Title:           title,
		}

		// Validate bindings
		validationErrors := ValidateBindings(exportDevice)
		allValidationErrors = append(allValidationErrors, validationErrors...)

		// Export to SVG
		if err := ExportToSVG(exportDevice, outputDir); err != nil {
			fmt.Printf("  ✗ Error exporting %s: %v\n", representativeName, err)
		} else {
			exportCount++
		}
	}

	// Display validation errors
	DisplayValidationErrors(allValidationErrors)

	fmt.Printf("\n✓ %d diagram(s) generated from CSV\n", exportCount)
	return nil
}

// parseSimulatorType converts simulator string to SimulationType
func parseSimulatorType(simulator string) common.SimulationType {
	switch strings.ToLower(simulator) {
	case "dcs_world", "dcs", "dcsworld":
		return common.DCSWorld
	case "il2_sturmovik", "il2", "il-2":
		return common.IL2Sturmovik
	case "il2_korea", "il2-korea", "korea":
		return common.IL2Korea
	default:
		return common.DCSWorld
	}
}

// findDeviceGUID searches for a device GUID in config by device name
func findDeviceGUID(deviceName string, config *common.Config) string {
	if config == nil {
		return ""
	}

	// Search in device_mappings
	for _, mapping := range config.DeviceMappings {
		if mapping.DeviceName == deviceName {
			return mapping.DeviceGUID
		}
	}

	return ""
}

// csvRowToBinding converts a CSV row to a Binding
func csvRowToBinding(row csvRow, deviceGUID, deviceName string) *common.Binding {
	if row.action == "" || row.physicalInput == "" {
		return nil
	}

	// Parse input type and ID from Physical Input
	// Format: "BTN1", "Axis X", "POV 1_U"
	var inputType common.InputType
	var inputID string

	switch {
	case strings.HasPrefix(row.physicalInput, "BTN"):
		inputType = common.Button
		inputID = strings.TrimPrefix(row.physicalInput, "BTN")
	case strings.HasPrefix(row.physicalInput, "Axis "):
		inputType = common.Axis
		inputID = strings.TrimPrefix(row.physicalInput, "Axis ")
	case strings.HasPrefix(row.physicalInput, "POV "):
		inputType = common.Hat
		inputID = strings.TrimPrefix(row.physicalInput, "POV ")
	default:
		return nil
	}

	// Parse modifier number if present
	modifierNum := 0
	if row.modifierNum != "" {
		_, _ = fmt.Sscanf(row.modifierNum, "%d", &modifierNum)
	}

	binding := &common.Binding{
		DeviceGUID:    deviceGUID,
		DeviceName:    deviceName,
		InputType:     inputType,
		InputID:       inputID,
		Action:        row.action,
		Modifiers:     parseModifiers(row.modifier, row.modifierDevice),
		ModifierNum:   modifierNum,
		VirtualDevice: row.virtualDevice,
		VirtualInput:  row.virtualInput,
	}

	return binding
}

// parseModifiers converts modifier string to Modifier slice
// Format: "BTN105", "BTN105 + BTN106", "Shift", etc.
func parseModifiers(modifierText string, modifierDevice string) []common.Modifier {
	if modifierText == "" {
		return nil
	}

	// Split by " + " to get individual modifier keys
	modifierKeys := strings.Split(modifierText, " + ")

	// Convert to proper format (add JOY_ prefix if needed)
	var normalizedKeys []string
	for _, key := range modifierKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		// If it's a button number (BTNxx), add JOY_ prefix
		switch {
		case strings.HasPrefix(key, "BTN"):
			normalizedKeys = append(normalizedKeys, "JOY_"+key)
		case strings.HasPrefix(key, "GREMLINS_MODE_"), strings.HasPrefix(key, "JOY_"):
			normalizedKeys = append(normalizedKeys, key)
		default:
			// For other modifiers (Shift, Ctrl, etc.), add GREMLINS_MODE_ prefix
			normalizedKeys = append(normalizedKeys, "GREMLINS_MODE_"+key)
		}
	}

	if len(normalizedKeys) == 0 {
		return nil
	}

	return []common.Modifier{
		{
			Keys:       normalizedKeys,
			Action:     "", // Action not stored in CSV modifier column
			DeviceName: modifierDevice,
		},
	}
}

// hashString creates a simple hash of a string (for generating GUIDs)
func hashString(s string) uint32 {
	var hash uint32
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return hash
}
