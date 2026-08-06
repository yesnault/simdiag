package svg

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
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

// groupKey identifies one generated diagram: all rows sharing a simulator, a module
// and a template are merged into a single SVG.
type groupKey struct {
	simulator    string
	module       string
	templatePath string
}

// resolveDeviceGUID determines the GUID to use for a row's physical device.
// The GUID stored in the CSV is preferred (round-trip fidelity); failing that the
// device is looked up by name in the config, and as a last resort a stable
// synthetic GUID is derived from the device name.
func resolveDeviceGUID(row csvRow, config *common.Config) string {
	if row.physicalDeviceGUID != "" {
		return row.physicalDeviceGUID
	}
	if guid := findDeviceGUID(row.physicalDevice, config); guid != "" {
		return guid
	}
	return fmt.Sprintf("%08x-0000-0000-0000-000000000000", hashString(row.physicalDevice))
}

// validateGenerationConfig checks that the config carries everything SVG
// generation needs.
func validateGenerationConfig(config *common.Config) error {
	switch {
	case config == nil:
		return fmt.Errorf("config is required")
	case config.TemplatesDirectory == "":
		return fmt.Errorf("templates directory not configured")
	case config.OutputDirectory == "":
		return fmt.Errorf("output directory not configured")
	}
	return nil
}

// readCSVGroups reads the export and buckets its rows by simulator, module and
// template. It also returns the keys in first-appearance order, so that diagram
// generation does not depend on map iteration order.
func readCSVGroups(csvPath string) (map[groupKey][]csvRow, []groupKey, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	header, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("error reading CSV header: %w", err)
	}

	colIndices, err := columnIndices(header)
	if err != nil {
		return nil, nil, err
	}

	groups := make(map[groupKey][]csvRow)
	var groupOrder []groupKey

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("error reading CSV row: %w", err)
		}

		row := rowFromRecord(record, colIndices)

		// Virtual-only bindings and rows without a template produce no diagram
		if row.physicalDevice == "" || row.templatePath == "" {
			continue
		}

		// Group by template, not by device: this merges bindings from several
		// devices that share the same template.
		key := groupKey{simulator: row.simulator, module: row.module, templatePath: row.templatePath}
		if _, seen := groups[key]; !seen {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], row)
	}

	return groups, groupOrder, nil
}

// columnIndices maps each expected column name to its position in the header,
// failing when a required column is absent.
func columnIndices(header []string) (map[string]int, error) {
	colIndices := make(map[string]int, len(header))
	for i, col := range header {
		colIndices[col] = i
	}

	for _, col := range simdiagcsv.AllColumns {
		if _, exists := colIndices[col]; !exists {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	return colIndices, nil
}

// rowFromRecord maps a raw CSV record onto the typed row struct.
func rowFromRecord(record []string, colIndices map[string]int) csvRow {
	return csvRow{
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
}

// collectGroupDevices builds the device map for a group and returns the device
// representing it. The representative comes from row order rather than map
// iteration, so that repeated runs produce the same diagram title.
func collectGroupDevices(rows []csvRow, config *common.Config) (map[string]*common.Device, *common.Device) {
	deviceMap := make(map[string]*common.Device)
	var representative *common.Device

	for _, row := range rows {
		if row.physicalDevice == "" {
			continue
		}

		deviceGUID := resolveDeviceGUID(row, config)
		if _, exists := deviceMap[deviceGUID]; !exists {
			deviceMap[deviceGUID] = &common.Device{
				GUID:      deviceGUID,
				Name:      row.physicalDevice,
				IsVirtual: false,
			}
		}

		if representative == nil {
			representative = deviceMap[deviceGUID]
		}
	}

	return deviceMap, representative
}

// buildExportDevice assembles the export device for one group of CSV rows.
// It returns nil when the group has no usable template or no physical device.
func buildExportDevice(key groupKey, rows []csvRow, config *common.Config) *common.ExportDevice {
	if len(rows) == 0 {
		return nil
	}

	simType := parseSimulatorType(key.simulator)

	absoluteTemplatePath := common.MakeAbsolutePath(key.templatePath, config.TemplatesDirectory)
	template, err := common.LoadTemplate(absoluteTemplatePath)
	if err != nil {
		fmt.Printf("  ⚠ Error loading template %s: %v\n", key.templatePath, err)
		return nil
	}

	deviceMap, representative := collectGroupDevices(rows, config)
	if representative == nil {
		return nil
	}

	// Merge the bindings of every device sharing this template
	profile := &common.Profile{
		Name:     representative.Name,
		SimType:  simType,
		Module:   key.module,
		Devices:  deviceMap,
		Bindings: make([]common.Binding, 0, len(rows)),
	}
	for _, row := range rows {
		if binding := csvRowToBinding(row, resolveDeviceGUID(row, config), row.physicalDevice); binding != nil {
			profile.Bindings = append(profile.Bindings, *binding)
		}
	}

	return &common.ExportDevice{
		Device:          representative,
		Template:        template,
		Profile:         profile,
		OutputDirectory: filepath.Join(config.OutputDirectory, common.OutputSubdir(simType, key.module)),
		SimulatorName:   simType.GetConfigKey(),
		SimdiagVersion:  common.SimdiagVersion,
		Title:           common.ExportTitle(simType, key.module, representative.Name),
	}
}

// GenerateSVGFromCSV reads a CSV file and generates SVG/PNG diagrams from it
func GenerateSVGFromCSV(csvPath string, config *common.Config) error {
	if err := validateGenerationConfig(config); err != nil {
		return err
	}

	groups, groupOrder, err := readCSVGroups(csvPath)
	if err != nil {
		return err
	}

	fmt.Printf("\nGenerating diagrams from CSV (%d groups)...\n", len(groups))

	// Process each group, in the order the groups first appeared in the CSV
	exportCount := 0
	var allValidationErrors []common.ValidationError
	for _, key := range groupOrder {
		exportDevice := buildExportDevice(key, groups[key], config)
		if exportDevice == nil {
			continue
		}

		allValidationErrors = append(allValidationErrors, ValidateBindings(exportDevice)...)

		if err := ExportToSVG(exportDevice, exportDevice.OutputDirectory); err != nil {
			fmt.Printf("  ✗ Error exporting %s: %v\n", exportDevice.Device.Name, err)
			continue
		}
		exportCount++
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

	inputType, inputID, ok := common.ParseInputText(row.physicalInput)
	if !ok {
		return nil
	}

	// Parse modifier number if present; an unparseable value simply means "none"
	modifierNum, _ := strconv.Atoi(row.modifierNum)

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
