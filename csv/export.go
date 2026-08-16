package csv

import (
	"cmp"
	stdcsv "encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"simdiag/common"
)

var (
	vJoyGUIDPattern   = regexp.MustCompile(`(?i)vjoy\s+device\s+([a-f0-9]{8})`)
	vJoyNumberPattern = regexp.MustCompile(`#(\d+)`)
)

// dedupKey identifies a unique exported row.
//
// Physical device and input are part of the key so that two different physical
// buttons emitting the same virtual keys (e.g. Joystick BTN5 and Throttle BTN12
// both sending LShift+G) are not collapsed together. Module is part of the key so
// that SRS and OpenKneeboard bindings survive across modules. The modifier is
// deliberately absent: one physical button used with different modifiers still
// produces a single row.
type dedupKey struct {
	action         string
	virtualDevice  string // normalized through normalizeVJoyName
	virtualInput   string
	physicalDevice string
	physicalInput  string
	module         string
}

// virtualRef addresses a virtual control: a normalized virtual device plus input.
type virtualRef struct {
	device string
	input  string
}

// physicalMapping records the physical control sitting behind a virtual input, so
// that every action bound to that virtual input can inherit it.
type physicalMapping struct {
	device     string
	input      string
	deviceGUID string
	modifier   string
}

// deviceMeta holds the per-ExportDevice values shared by all of its rows.
type deviceMeta struct {
	simulator    string
	module       string
	templatePath string
}

// ExportToCSV exports all bindings to a CSV file
func ExportToCSV(exportDevices []*common.ExportDevice, outputPath string, config *common.Config) error {
	if len(exportDevices) == 0 {
		return nil
	}

	rowMap, physicalMappings := collectRows(exportDevices, config)
	propagatePhysicalMappings(rowMap, physicalMappings)

	return writeRows(outputPath, rowMap)
}

// collectRows turns every binding into a deduplicated CSV row, and indexes which
// physical control drives each virtual input.
func collectRows(exportDevices []*common.ExportDevice, config *common.Config) (map[dedupKey]*csvRowData, map[virtualRef]*physicalMapping) {
	rowMap := make(map[dedupKey]*csvRowData)
	physicalMappings := make(map[virtualRef]*physicalMapping)

	for _, exportDevice := range exportDevices {
		meta := deviceMeta{
			simulator:    exportDevice.Profile.SimType.GetConfigKey(),
			module:       common.ModuleKey(exportDevice.Profile.SimType, exportDevice.Profile.Module),
			templatePath: templatePathFor(exportDevice, config),
		}

		// Switch/Modifier bindings are included so diagrams can be regenerated
		// from the CSV with their mode-switching information intact.
		for _, binding := range exportDevice.Profile.Bindings {
			row, key, ok := buildRow(binding, exportDevice, meta)
			if !ok {
				continue
			}

			recordPhysicalMapping(physicalMappings, key, row)
			mergeRow(rowMap, key, row)
		}
	}

	return rowMap, physicalMappings
}

// templatePathFor returns the export device's template path, relative to the
// configured templates directory when possible.
func templatePathFor(exportDevice *common.ExportDevice, config *common.Config) string {
	if exportDevice.Template == nil || exportDevice.Template.FilePath == "" {
		return ""
	}
	if config == nil || config.TemplatesDirectory == "" {
		return exportDevice.Template.FilePath
	}
	return common.MakeRelativePath(exportDevice.Template.FilePath, config.TemplatesDirectory)
}

// buildRow converts one binding into a CSV row and its deduplication key.
// It reports false for bindings that do not belong in the export (keyboard).
func buildRow(binding common.Binding, exportDevice *common.ExportDevice, meta deviceMeta) (*csvRowData, dedupKey, bool) {
	// Only joystick/controller bindings are exported
	if strings.EqualFold(binding.DeviceName, "Keyboard") {
		return nil, dedupKey{}, false
	}

	// Keep every action on a single CSV line; quoting is handled by the writer
	action := strings.TrimSpace(strings.ReplaceAll(binding.DisplayText(), "\n", " "))

	modifierText, modifierDeviceName := formatModifiers(binding, exportDevice.Profile)

	modifierNum := ""
	if binding.ModifierNum > 0 {
		// Already assigned by AssignModifierNumbers
		modifierNum = fmt.Sprintf("%d", binding.ModifierNum)
	}

	physical, virtual := splitPhysicalVirtual(binding)

	row := &csvRowData{
		simulator:          meta.simulator,
		module:             meta.module,
		action:             action,
		modifier:           modifierText,
		modifierDevice:     modifierDeviceName,
		modifierNum:        modifierNum,
		physicalDevice:     physical.device,
		physicalInput:      physical.input,
		physicalDeviceGUID: physical.deviceGUID,
		virtualDevice:      virtual.device,
		virtualInput:       virtual.input,
		templateKey:        common.TemplateKeyFor(binding.InputType, binding.InputID),
		templatePath:       meta.templatePath,
	}

	key := dedupKey{
		action:         action,
		virtualDevice:  normalizedVJoyDevice(virtual.device),
		virtualInput:   virtual.input,
		physicalDevice: physical.device,
		physicalInput:  physical.input,
		module:         meta.module,
	}

	return row, key, true
}

// formatModifiers renders a binding's modifiers as they appear in the CSV
// ("BTN105 + Shift") and returns the device owning the first joystick modifier.
func formatModifiers(binding common.Binding, profile *common.Profile) (text, deviceName string) {
	if len(binding.Modifiers) == 0 {
		return "", ""
	}

	modifierKeys := make([]string, 0, len(binding.Modifiers))
	for _, mod := range binding.Modifiers {
		for _, key := range mod.Keys {
			// The first joystick modifier names the device shown in the legend
			if deviceName == "" && strings.HasPrefix(key, "JOY_BTN") {
				if modInfo, exists := profile.ModifierDeviceMap[key]; exists {
					deviceName = modInfo.DeviceName
				}
			}

			// "JOY_BTN105" -> "BTN105", "GREMLINS_MODE_Shift" -> "Shift"
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

	return strings.Join(modifierKeys, " + "), deviceName
}

// physicalRef holds the physical columns of a row.
type physicalRef struct {
	device     string
	input      string
	deviceGUID string
}

// splitPhysicalVirtual routes a binding into the physical or the virtual columns.
// vJoy devices are virtual controls, so they land in the virtual columns and leave
// the physical ones empty until propagatePhysicalMappings fills them in.
func splitPhysicalVirtual(binding common.Binding) (physicalRef, virtualRef) {
	inputText := common.FormatInputText(binding.InputType, binding.InputID)

	virtual := virtualRef{device: binding.VirtualDevice, input: binding.VirtualInput}

	// Normalize key order so that "F + LShift" and "LShift + F" compare equal
	if virtual.device == "Keyboard" && virtual.input != "" {
		keys := common.NormalizeKeyOrder(strings.Split(virtual.input, " + "))
		virtual.input = strings.Join(keys, " + ")
	}

	if !strings.Contains(strings.ToLower(binding.DeviceName), "vjoy") {
		return physicalRef{
			device:     binding.DeviceName,
			input:      inputText,
			deviceGUID: binding.DeviceGUID,
		}, virtual
	}

	// Normalize the name to "vJoy Device <GUID[:8]>" for clarity
	virtual.device = binding.DeviceName
	if shortGUID := common.NormalizeGUIDShort(binding.DeviceGUID); shortGUID != "" {
		virtual.device = "vJoy Device " + shortGUID
	}
	virtual.input = inputText

	return physicalRef{}, virtual
}

// recordPhysicalMapping remembers the physical control behind a virtual input, the
// first time that virtual input is seen with one.
func recordPhysicalMapping(physicalMappings map[virtualRef]*physicalMapping, key dedupKey, row *csvRowData) {
	if row.physicalDevice == "" || row.virtualInput == "" {
		return
	}

	ref := virtualRef{device: key.virtualDevice, input: row.virtualInput}
	if _, exists := physicalMappings[ref]; exists {
		return
	}

	physicalMappings[ref] = &physicalMapping{
		device:     row.physicalDevice,
		input:      row.physicalInput,
		deviceGUID: row.physicalDeviceGUID,
		modifier:   row.modifier,
	}
}

// mergeRow stores row under key, keeping the first one seen.
//
// It used to prefer the variant carrying physical device information, and a real
// action label over a bare "TARGET" one. Neither preference could ever apply:
// dedupKey holds action and physicalDevice, so two rows colliding on a key have
// the same values for both. The conditions were dead, and dead code that reads
// like a deliberate policy is worse than none.
func mergeRow(rowMap map[dedupKey]*csvRowData, key dedupKey, row *csvRowData) {
	if _, exists := rowMap[key]; !exists {
		rowMap[key] = row
	}
}

// propagatePhysicalMappings fills in the physical columns of rows that only know
// their virtual control. When Gremlins maps Throttle BTN8 to vJoy2 BTN8, every
// action on vJoy2 BTN8 gets the throttle's device information.
func propagatePhysicalMappings(rowMap map[dedupKey]*csvRowData, physicalMappings map[virtualRef]*physicalMapping) {
	for _, row := range rowMap {
		if row.physicalDevice != "" || row.virtualInput == "" {
			continue
		}

		ref := virtualRef{device: normalizedVJoyDevice(row.virtualDevice), input: row.virtualInput}
		mapping, exists := physicalMappings[ref]
		if !exists {
			continue
		}

		row.physicalDevice = mapping.device
		row.physicalInput = mapping.input
		row.physicalDeviceGUID = mapping.deviceGUID

		// Also propagate the modifier from Gremlins if the row doesn't have one
		if row.modifier == "" && mapping.modifier != "" {
			row.modifier = mapping.modifier
		}
	}
}

// writeRows writes the header and every unique row to outputPath, in a stable
// order so that repeated exports are byte-identical.
func writeRows(outputPath string, rowMap map[dedupKey]*csvRowData) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating CSV file: %w", err)
	}
	defer file.Close()

	writer := stdcsv.NewWriter(file)
	if err := writer.Write(AllColumns); err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	for _, key := range sortedKeys(rowMap) {
		if err := writer.Write(rowMap[key].fields()); err != nil {
			return fmt.Errorf("error writing CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("error writing CSV: %w", err)
	}

	return nil
}

// sortedKeys returns the row keys in a deterministic order.
func sortedKeys(rowMap map[dedupKey]*csvRowData) []dedupKey {
	keys := make([]dedupKey, 0, len(rowMap))
	for key := range rowMap {
		keys = append(keys, key)
	}

	slices.SortFunc(keys, func(a, b dedupKey) int {
		return cmp.Or(
			cmp.Compare(a.module, b.module),
			cmp.Compare(a.physicalDevice, b.physicalDevice),
			cmp.Compare(a.physicalInput, b.physicalInput),
			cmp.Compare(a.action, b.action),
			cmp.Compare(a.virtualDevice, b.virtualDevice),
			cmp.Compare(a.virtualInput, b.virtualInput),
		)
	})

	return keys
}

// normalizedVJoyDevice collapses the various spellings of a vJoy device name to a
// single identifier, leaving non-vJoy names untouched.
func normalizedVJoyDevice(name string) string {
	if !strings.Contains(strings.ToLower(name), "vjoy") {
		return name
	}
	return normalizeVJoyName(name)
}

// normalizeVJoyName extracts a consistent identifier from vJoy device names
// "vJoy Device" -> "vJoy"
// "vJoy Device #1" -> "vJoy_1"
// "vJoy Device #2" -> "vJoy_2"
// "vJoy Device abc12345" -> "vJoy_abc12345"
func normalizeVJoyName(name string) string {
	// Try to extract GUID prefix from "vJoy Device abc12345"
	if matches := vJoyGUIDPattern.FindStringSubmatch(name); len(matches) >= 2 {
		return "vJoy_" + strings.ToLower(matches[1])
	}

	// Try to extract number from "vJoy Device #X"
	if matches := vJoyNumberPattern.FindStringSubmatch(name); len(matches) >= 2 {
		return "vJoy_" + matches[1]
	}

	return "vJoy"
}
