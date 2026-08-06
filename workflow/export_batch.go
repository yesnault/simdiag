package workflow

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"simdiag/common"
	"simdiag/csv"
	"simdiag/svg"
)

// EnrichmentFunc is a function that adds bindings from external tools to an ExportDevice
type EnrichmentFunc func(exportDevice *common.ExportDevice, fullProfile *common.Profile, config *common.Config)

// exportWorkflowBatch handles SVG export in batch mode (non-interactive)
func exportWorkflowBatch(profiles *common.ProfileCollection, simType common.SimulationType, enrichmentFuncs []EnrichmentFunc, filter string) *exportResult {
	// Load existing configuration
	config, err := common.LoadConfig()
	if err != nil {
		fmt.Printf("⚠ Unable to load config: %v\n", err)
		fmt.Println("No mappings available. Export cancelled.")
		return &exportResult{}
	}

	// For DCS World, process each module separately
	if simType == common.DCSWorld {
		return exportDCSBatch(profiles, config, enrichmentFuncs, filter)
	}
	// For other sims (IL-2), use flat structure
	return exportNonModularBatch(profiles, config, simType, enrichmentFuncs)
}

// ExportAllSimulatorsBatchWithInterfaces processes all configured simulators in batch mode using interfaces
func ExportAllSimulatorsBatchWithInterfaces(parsers map[common.SimulationType]common.SimulatorParser, enrichers []common.BindingEnricher, filter string, noSVG bool) {
	// Load existing configuration
	config, err := common.LoadConfig()
	if err != nil {
		fmt.Printf("⚠ Unable to load config: %v\n", err)
		fmt.Println("No configuration available. Export cancelled.")
		return
	}

	if filter != "" {
		fmt.Printf("Batch mode: processing filtered simulators/modules (filter: %s)\n", filter)
	} else {
		fmt.Println("Batch mode: processing all configured simulators")
	}
	fmt.Println()

	// Collect all export devices from all simulators
	var allExportDevices []*common.ExportDevice

	// Process each configured simulator
	for simType, parser := range parsers {
		result := processSimulator(parser, simType, config, enrichers, filter)
		if result != nil {
			allExportDevices = append(allExportDevices, result.devices...)
		}
	}

	// Export CSV and optionally generate SVG from CSV
	finishBatchExport(allExportDevices, config, noSVG)

	fmt.Println("\n========================================")
	fmt.Println("=== Batch processing complete ===")
	fmt.Println("========================================")
}

// processSimulator processes a single simulator type
func processSimulator(parser common.SimulatorParser, simType common.SimulationType, config *common.Config, enrichers []common.BindingEnricher, filter string) *exportResult {
	if !shouldProcessSimulator(simType, config, filter) {
		return nil
	}

	profiles, err := parseSimulatorProfiles(parser, simType, config)
	if err != nil {
		fmt.Printf("⚠ Error parsing %s: %v\n", parser.GetName(), err)
		return nil
	}

	displayProfileInfo(parser, simType, profiles)

	return exportWorkflowBatch(profiles, simType, toEnrichmentFuncs(enrichers), filter)
}

// shouldProcessSimulator determines if a simulator should be processed
func shouldProcessSimulator(simType common.SimulationType, config *common.Config, filter string) bool {
	simConfig := config.GetSimulatorConfig(simType)
	if simConfig == nil {
		return false
	}

	// Apply filter at simulator level
	if filter != "" && !matchesFilter(string(simType), filter) && simType != common.DCSWorld {
		return false
	}

	// Check if simulator has configuration
	if !hasSimulatorConfig(simType, simConfig) {
		return false
	}

	// Verify required directories
	if config.TemplatesDirectory == "" {
		fmt.Println("⚠ Global templates directory not configured. Skipping.")
		return false
	}
	if config.OutputDirectory == "" {
		fmt.Println("⚠ Output directory not configured. Skipping.")
		return false
	}

	return true
}

// hasSimulatorConfig checks if a simulator has valid configuration
func hasSimulatorConfig(simType common.SimulationType, simConfig *common.SimulatorConfig) bool {
	if simType == common.DCSWorld {
		return len(simConfig.Modules) > 0
	}
	return simConfig.IL2InputPath != ""
}

// parseSimulatorProfiles parses simulator configuration files
func parseSimulatorProfiles(parser common.SimulatorParser, simType common.SimulationType, config *common.Config) (*common.ProfileCollection, error) {
	fmt.Println("\n========================================")
	fmt.Printf("=== Processing %s ===\n", parser.GetName())
	fmt.Println("========================================")

	configPath := common.GetConfigPath(config, simType, true)
	return parser.Parse(configPath)
}

// displayProfileInfo displays information about parsed profiles
func displayProfileInfo(parser common.SimulatorParser, simType common.SimulationType, profiles *common.ProfileCollection) {
	profileNames := make([]string, 0)
	for _, p := range profiles.Profiles {
		if simType == common.DCSWorld && p.Module != "" {
			profileNames = append(profileNames, p.Module)
		} else if p.Name != "" {
			profileNames = append(profileNames, p.Name)
		}
	}
	fmt.Printf("\n%s: Found %d profiles: %s\n", parser.GetName(), len(profileNames), strings.Join(profileNames, ", "))
}

// matchesFilter checks if a name matches the filter (case-insensitive partial match)
func matchesFilter(name, filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(filter))
}

// exportResult holds the result of a batch export
type exportResult struct {
	devices []*common.ExportDevice
}

// toEnrichmentFuncs converts BindingEnricher slice to EnrichmentFunc slice
func toEnrichmentFuncs(enrichers []common.BindingEnricher) []EnrichmentFunc {
	funcs := make([]EnrichmentFunc, len(enrichers))
	for i, enricher := range enrichers {
		e := enricher // Capture for closure
		funcs[i] = func(exportDevice *common.ExportDevice, fullProfile *common.Profile, config *common.Config) {
			e.Enrich(exportDevice, fullProfile, config)
		}
	}
	return funcs
}

// batchExportContext holds all data needed for the common batch export loop.
type batchExportContext struct {
	devices         map[string]*common.Device
	deviceTemplates map[string]*common.Template
	profiles        []*common.Profile // source profiles for binding lookup
	fullProfile     *common.Profile   // all keyboard/virtual bindings for Gremlins/TARGET enrichment
	simType         common.SimulationType
	moduleName      string
	config          *common.Config
	enrichmentFuncs []EnrichmentFunc // functions to enrich bindings (Gremlins, TARGET, OpenKneeboard)
}

// runBatchExport runs the common export loop: group by template, merge bindings,
// add enrichments (SRS/Gremlins/TARGET/OpenKneeboard), and prepare devices for CSV export.
// Returns the number of devices processed and all export devices (for CSV).
func runBatchExport(ctx *batchExportContext) (int, []*common.ExportDevice) {
	exportCount := 0
	var allExportDevices []*common.ExportDevice

	// Group devices by template
	templateGroups := common.GroupDevicesByTemplate(ctx.deviceTemplates)

	for _, deviceGUIDs := range templateGroups {
		template := ctx.deviceTemplates[deviceGUIDs[0]]
		if template == nil {
			continue
		}

		exportDevice := createExportDeviceForGroup(deviceGUIDs, template, ctx)
		enrich(exportDevice, ctx)

		allExportDevices = append(allExportDevices, exportDevice)
		exportCount++
	}

	// Handle devices without templates (create individual export devices)
	for guid, device := range ctx.devices {
		if device.IsVirtual {
			continue
		}

		// Skip if device already has a template
		if _, hasTemplate := ctx.deviceTemplates[guid]; hasTemplate {
			continue
		}

		exportDevice := createExportDeviceWithoutTemplate(guid, device, ctx)
		enrich(exportDevice, ctx)

		allExportDevices = append(allExportDevices, exportDevice)
		exportCount++
	}

	return exportCount, allExportDevices
}

// createExportDeviceForGroup creates an ExportDevice for a group of devices sharing a template
func createExportDeviceForGroup(deviceGUIDs []string, template *common.Template, ctx *batchExportContext) *common.ExportDevice {
	mergedProfile := createMergedProfile(ctx)
	mergeDeviceBindings(deviceGUIDs, ctx, mergedProfile)
	representativeDevice := ctx.devices[deviceGUIDs[0]]

	return &common.ExportDevice{
		Device:   representativeDevice,
		Template: template,
		Profile:  mergedProfile,
	}
}

// createMergedProfile creates a profile combining bindings from all devices in the group
func createMergedProfile(ctx *batchExportContext) *common.Profile {
	// Calculate profile name
	var profileName string
	switch {
	case ctx.simType == common.DCSWorld && ctx.moduleName != "":
		profileName = fmt.Sprintf("DCS World / %s", strings.ToUpper(ctx.moduleName))
	case len(ctx.profiles) > 0:
		profileName = ctx.profiles[0].Name
	default:
		profileName = "External Bindings"
	}

	mergedProfile := &common.Profile{
		Name:              profileName,
		SimType:           ctx.simType,
		Module:            ctx.moduleName,
		Devices:           make(map[string]*common.Device),
		Bindings:          make([]common.Binding, 0),
		ModifierDeviceMap: make(map[string]common.ModifierInfo),
	}

	// Copy ModifierDeviceMap from fullProfile if available
	if ctx.fullProfile != nil && ctx.fullProfile.ModifierDeviceMap != nil {
		for key, value := range ctx.fullProfile.ModifierDeviceMap {
			mergedProfile.ModifierDeviceMap[key] = value
		}
	}

	return mergedProfile
}

// mergeDeviceBindings registers each device of the group on the merged profile and
// copies over the bindings that belong to it.
func mergeDeviceBindings(deviceGUIDs []string, ctx *batchExportContext, mergedProfile *common.Profile) {
	for _, deviceGUID := range deviceGUIDs {
		device := ctx.devices[deviceGUID]
		if device == nil {
			continue
		}

		mergedProfile.Devices[deviceGUID] = device

		// Add bindings for this device from all source profiles
		for _, profile := range ctx.profiles {
			for _, binding := range profile.Bindings {
				// Use partial GUID matching to handle IL-2 vs TARGET GUID format differences
				if common.MatchGUIDPartial(binding.DeviceGUID, deviceGUID) {
					mergedProfile.Bindings = append(mergedProfile.Bindings, binding)
				}
			}
		}
	}

	// If no bindings found, create empty profile for external bindings
	if len(mergedProfile.Bindings) == 0 && ctx.moduleName == "" {
		mergedProfile.Name = "External Bindings"
	}
}

// displayDeviceListCSVMode displays a simple list of devices for CSV-only mode
func displayDeviceListCSVMode(profiles []*common.Profile) {
	// Collect all device names (deduplicated)
	deviceNames := make(map[string]bool)
	for _, profile := range profiles {
		for _, device := range profile.Devices {
			if !device.IsVirtual {
				deviceNames[device.Name] = true
			}
		}
	}

	// Display sorted list
	for _, name := range slices.Sorted(maps.Keys(deviceNames)) {
		fmt.Printf("  → %s\n", name)
	}
}

// createExportDeviceWithoutTemplate creates an ExportDevice without template
func createExportDeviceWithoutTemplate(deviceGUID string, device *common.Device, ctx *batchExportContext) *common.ExportDevice {
	mergedProfile := createMergedProfile(ctx)

	// Add bindings for this device from all source profiles
	for _, profile := range ctx.profiles {
		for _, binding := range profile.Bindings {
			// Use partial GUID matching to handle IL-2 vs TARGET GUID format differences
			if common.MatchGUIDPartial(binding.DeviceGUID, deviceGUID) {
				mergedProfile.Bindings = append(mergedProfile.Bindings, binding)
			}
		}
	}

	mergedProfile.Devices[deviceGUID] = device

	return &common.ExportDevice{
		Device:   device,
		Template: nil,
		Profile:  mergedProfile,
	}
}

// enrich applies enrichment functions to add bindings from external tools
func enrich(exportDevice *common.ExportDevice, ctx *batchExportContext) {
	// Add enrichment bindings (Gremlins, TARGET, OpenKneeboard, SRS, etc.)
	for _, enrichFunc := range ctx.enrichmentFuncs {
		enrichFunc(exportDevice, ctx.fullProfile, ctx.config)
	}
}

// finishBatchExport exports CSV and optionally generates SVG from CSV.
func finishBatchExport(allExportDevices []*common.ExportDevice, config *common.Config, noSVG bool) {
	if len(allExportDevices) == 0 {
		return
	}
	// Assign modifier numbers to all bindings before export
	common.AssignModifierNumbers(allExportDevices)

	csvPath := filepath.Join(config.OutputDirectory, "export.csv")
	if err := csv.ExportToCSV(allExportDevices, csvPath, config); err != nil {
		fmt.Printf("\n⚠ CSV export failed: %v\n", err)
		return
	}

	fmt.Printf("\n✓ CSV exported: %s\n", csvPath)

	// Generate SVG/PNG from CSV (skip if --no-svg flag is set)
	if noSVG {
		return
	}

	if err := svg.GenerateSVGFromCSV(csvPath, config); err != nil {
		fmt.Printf("⚠ SVG generation from CSV failed: %v\n", err)
	}
}

// exportModule exports diagrams for a set of profiles (used by both DCS and IL-2)
func exportModule(profiles []*common.Profile, simType common.SimulationType, moduleName string, config *common.Config, enrichmentFuncs []EnrichmentFunc) (int, []*common.ExportDevice) {
	// Get devices
	devices := collectDevicesFromProfiles(profiles)

	// Load template paths only (no SVG content)
	deviceTemplates := common.LoadDeviceTemplatePathsOnly(devices, config)

	// Build export context
	ctx := buildExportContext(profiles, simType, moduleName, config, devices, deviceTemplates, enrichmentFuncs)

	return runBatchExport(ctx)
}

// collectDevicesFromProfiles collects all devices from profiles and marks virtual devices
func collectDevicesFromProfiles(profiles []*common.Profile) map[string]*common.Device {
	devices := make(map[string]*common.Device)
	for _, profile := range profiles {
		maps.Copy(devices, profile.Devices)
	}
	common.MarkVirtualDevicesInMap(devices)
	return devices
}

// buildExportContext creates a batch export context from the given parameters
func buildExportContext(profiles []*common.Profile, simType common.SimulationType, moduleName string, config *common.Config, devices map[string]*common.Device, deviceTemplates map[string]*common.Template, enrichmentFuncs []EnrichmentFunc) *batchExportContext {
	fullProfile := buildFullProfile(profiles, simType, moduleName)

	return &batchExportContext{
		devices:         devices,
		deviceTemplates: deviceTemplates,
		profiles:        profiles,
		fullProfile:     fullProfile,
		simType:         simType,
		moduleName:      moduleName,
		config:          config,
		enrichmentFuncs: enrichmentFuncs,
	}
}

// buildFullProfile builds a profile with keyboard/virtual device bindings for enrichment
func buildFullProfile(profiles []*common.Profile, simType common.SimulationType, moduleName string) *common.Profile {
	var profileName string
	if len(profiles) > 0 {
		if moduleName != "" {
			profileName = moduleName
		} else {
			profileName = profiles[0].Name
		}
	} else {
		profileName = "External Bindings"
	}

	fullProfile := &common.Profile{
		Name:              profileName,
		SimType:           simType,
		Module:            moduleName,
		Bindings:          make([]common.Binding, 0),
		ModifierDeviceMap: make(map[string]common.ModifierInfo),
	}

	// Copy ModifierDeviceMap from all profiles
	for _, profile := range profiles {
		if profile.ModifierDeviceMap != nil {
			maps.Copy(fullProfile.ModifierDeviceMap, profile.ModifierDeviceMap)
		}
	}

	// Collect keyboard/virtual device bindings
	for _, profile := range profiles {
		for _, binding := range profile.Bindings {
			if binding.DeviceGUID == "keyboard" || common.IsVirtualDevice(binding.DeviceName) || common.IsVirtualDeviceGUID(binding.DeviceGUID) {
				fullProfile.Bindings = append(fullProfile.Bindings, binding)
			}
		}
	}

	return fullProfile
}

// exportDCSBatch handles batch export for DCS World with modules
func exportDCSBatch(profiles *common.ProfileCollection, config *common.Config, enrichmentFuncs []EnrichmentFunc, filter string) *exportResult {
	systemProfiles, moduleProfiles := separateDCSProfiles(profiles)

	if len(moduleProfiles) == 0 {
		fmt.Println("⚠ No module profiles found. Export cancelled.")
		return &exportResult{}
	}

	result := exportDCSModules(moduleProfiles, systemProfiles, config, enrichmentFuncs, filter)

	fmt.Printf("\n=== Total: %d device(s) processed ===\n", result.totalExported)

	return &exportResult{
		devices: result.allExportDevices,
	}
}

// separateDCSProfiles separates system profiles from module profiles
func separateDCSProfiles(profiles *common.ProfileCollection) ([]*common.Profile, map[string][]*common.Profile) {
	systemProfiles := make([]*common.Profile, 0)
	moduleProfiles := make(map[string][]*common.Profile)

	for _, profile := range profiles.Profiles {
		if profile.Module == "" {
			systemProfiles = append(systemProfiles, profile)
		} else {
			moduleProfiles[profile.Module] = append(moduleProfiles[profile.Module], profile)
		}
	}

	return systemProfiles, moduleProfiles
}

// dcsExportResult holds the result of exporting DCS modules
type dcsExportResult struct {
	totalExported    int
	allExportDevices []*common.ExportDevice
}

// exportDCSModules exports all DCS modules
func exportDCSModules(moduleProfiles map[string][]*common.Profile, systemProfiles []*common.Profile, config *common.Config, enrichmentFuncs []EnrichmentFunc, filter string) *dcsExportResult {
	result := &dcsExportResult{
		allExportDevices: make([]*common.ExportDevice, 0),
	}

	for moduleName, moduleProfilesList := range moduleProfiles {
		if filter != "" && !matchesFilter(moduleName, filter) {
			continue
		}

		exportDCSModule(moduleName, moduleProfilesList, systemProfiles, config, enrichmentFuncs, result)
	}

	return result
}

// exportDCSModule exports a single DCS module
func exportDCSModule(moduleName string, moduleProfilesList []*common.Profile, systemProfiles []*common.Profile, config *common.Config, enrichmentFuncs []EnrichmentFunc, result *dcsExportResult) {
	fmt.Printf("\n=== Module: %s ===\n", moduleName)

	// Combine module and system profiles
	allProfiles := make([]*common.Profile, 0, len(moduleProfilesList)+len(systemProfiles))
	allProfiles = append(allProfiles, moduleProfilesList...)
	allProfiles = append(allProfiles, systemProfiles...)

	// Show "Collecting bindings" section
	fmt.Printf("\n=== Collecting bindings ===\n")
	fmt.Println()
	displayDeviceListCSVMode(allProfiles)
	fmt.Println()

	// Export this module
	moduleExported, exportDevices := exportModule(allProfiles, common.DCSWorld, moduleName, config, enrichmentFuncs)
	result.allExportDevices = append(result.allExportDevices, exportDevices...)

	fmt.Printf("✓ %d device(s) processed\n", moduleExported)

	result.totalExported += moduleExported
}

// exportNonModularBatch handles batch export for non-modular simulators (IL-2)
func exportNonModularBatch(profiles *common.ProfileCollection, config *common.Config, simType common.SimulationType, enrichmentFuncs []EnrichmentFunc) *exportResult {
	devices := common.GetAllDevicesFromProfiles(profiles)
	if len(devices) == 0 {
		fmt.Println("No device detected.")
		return &exportResult{}
	}

	// Show "Collecting bindings" section
	fmt.Println("\n=== Collecting bindings ===")
	fmt.Println()
	displayDeviceListCSVMode(profiles.Profiles)
	fmt.Println()

	// Export
	exportCount, allExportDevices := exportModule(profiles.Profiles, simType, "", config, enrichmentFuncs)

	fmt.Printf("✓ %d device(s) processed\n", exportCount)

	return &exportResult{
		devices: allExportDevices,
	}
}
