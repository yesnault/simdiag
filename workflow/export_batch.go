package workflow

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"simdiag/common"
	"simdiag/csv"
	"simdiag/svg"
)

// EnrichmentFunc is a function that adds bindings from external tools to an ExportDevice
type EnrichmentFunc func(exportDevice *common.ExportDevice, fullProfile *common.Profile, config *common.Config)

// SimulatorResult summarizes what a single simulator contributed to an export.
type SimulatorResult struct {
	Simulator string   `json:"simulator"`
	Modules   []string `json:"modules"`
	Devices   int      `json:"devices"`
	Bindings  int      `json:"bindings"`
}

// ExportSummary is the structured outcome of an export run. The batch CLI simply
// prints its progress as it goes; a GUI needs the same information as data.
type ExportSummary struct {
	Simulators []SimulatorResult `json:"simulators"`
	Warnings   []string          `json:"warnings"`
	Errors     []string          `json:"errors"`
	CSVPath    string            `json:"csvPath"`
	Devices    int               `json:"devices"`
	Bindings   int               `json:"bindings"`
	DurationMS int64             `json:"durationMs"`
	// ValidationErrors are bindings whose template has no matching key. They are
	// not failures: the diagram is produced, that binding just cannot be shown.
	ValidationErrors []common.ValidationError `json:"validationErrors"`
}

// warnf records a warning in the summary and echoes it to the progress output.
func (s *ExportSummary) warnf(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	s.Warnings = append(s.Warnings, msg)
	common.Printf("⚠ %s\n", msg)
}

// errorf records an error in the summary and echoes it to the progress output.
func (s *ExportSummary) errorf(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	s.Errors = append(s.Errors, msg)
	common.Printf("⚠ %s\n", msg)
}

// exportWorkflowBatch handles SVG export in batch mode (non-interactive)
func exportWorkflowBatch(ctx context.Context, config *common.Config, profiles *common.ProfileCollection, simType common.SimulationType, enrichmentFuncs []EnrichmentFunc, filter string) *exportResult {
	// For DCS World, process each module separately
	if simType == common.DCSWorld {
		return exportDCSBatch(ctx, profiles, config, enrichmentFuncs, filter)
	}
	// For other sims (IL-2), use flat structure
	return exportNonModularBatch(profiles, config, simType, enrichmentFuncs)
}

// ExportAll runs the full batch pipeline (Parse -> Enrich -> CSV -> SVG) against an
// already-loaded configuration and reports the outcome as data. This is the entry
// point for any caller that is not a terminal.
func ExportAll(ctx context.Context, config *common.Config, parsers map[common.SimulationType]common.SimulatorParser, enrichers []common.BindingEnricher, filter string, noSVG bool) (*ExportSummary, error) {
	if config == nil {
		return nil, fmt.Errorf("no configuration provided")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	started := time.Now()
	summary := &ExportSummary{}

	if filter != "" {
		common.Printf("Batch mode: processing filtered simulators/modules (filter: %s)\n", filter)
	} else {
		common.Println("Batch mode: processing all configured simulators")
	}
	common.Println()

	// Collect all export devices from all simulators
	var allExportDevices []*common.ExportDevice

	// Process each configured simulator, in a stable order so that the CSV and the
	// progress log do not depend on Go's map iteration order.
	for _, simType := range slices.Sorted(maps.Keys(parsers)) {
		if err := ctx.Err(); err != nil {
			summary.DurationMS = time.Since(started).Milliseconds()
			return summary, err
		}

		result := processSimulator(ctx, parsers[simType], simType, config, enrichers, filter, summary)
		if result != nil {
			allExportDevices = append(allExportDevices, result.devices...)
			summary.Simulators = append(summary.Simulators, summarizeSimulator(parsers[simType].GetName(), result.devices))
		}
	}

	summary.Devices = len(allExportDevices)
	for _, d := range allExportDevices {
		if d.Profile != nil {
			summary.Bindings += len(d.Profile.Bindings)
		}
	}

	// Export CSV and optionally generate SVG from CSV
	finishBatchExport(ctx, allExportDevices, config, noSVG, summary)

	common.Println("\n========================================")
	common.Println("=== Batch processing complete ===")
	common.Println("========================================")

	summary.DurationMS = time.Since(started).Milliseconds()
	return summary, ctx.Err()
}

// summarizeSimulator aggregates the export devices produced by one simulator.
func summarizeSimulator(name string, devices []*common.ExportDevice) SimulatorResult {
	res := SimulatorResult{Simulator: name, Devices: len(devices)}

	moduleSet := make(map[string]bool)
	for _, d := range devices {
		if d.Profile == nil {
			continue
		}
		res.Bindings += len(d.Profile.Bindings)
		if d.Profile.Module != "" {
			moduleSet[d.Profile.Module] = true
		}
	}
	res.Modules = slices.Sorted(maps.Keys(moduleSet))

	return res
}

// processSimulator processes a single simulator type
func processSimulator(ctx context.Context, parser common.SimulatorParser, simType common.SimulationType, config *common.Config, enrichers []common.BindingEnricher, filter string, summary *ExportSummary) *exportResult {
	if !shouldProcessSimulator(simType, config, filter, summary) {
		return nil
	}

	profiles, err := parseSimulatorProfiles(parser, simType, config)
	if err != nil {
		summary.errorf("Error parsing %s: %v", parser.GetName(), err)
		return nil
	}

	if err := ctx.Err(); err != nil {
		return nil
	}

	displayProfileInfo(parser, simType, profiles)

	return exportWorkflowBatch(ctx, config, profiles, simType, toEnrichmentFuncs(enrichers), filter)
}

// shouldProcessSimulator determines if a simulator should be processed
func shouldProcessSimulator(simType common.SimulationType, config *common.Config, filter string, summary *ExportSummary) bool {
	// Read without creating: a simulator the user never configured is not worth a
	// warning, whereas one configured incompletely is.
	simConfig := config.LookupSimulatorConfig(simType)
	if simConfig == nil {
		return false
	}

	// Apply filter at simulator level
	if filter != "" && !matchesFilter(string(simType), filter) && simType != common.DCSWorld {
		return false
	}

	// Check if simulator has configuration
	if !common.SimulatorIsConfigured(simType, simConfig) {
		summary.warnf("%s: %s is not configured. Skipping.", simType, common.RequiredPathSetting(simType))
		return false
	}

	// Verify required directories
	if config.TemplatesDirectory == "" {
		summary.warnf("Global templates directory not configured. Skipping.")
		return false
	}
	if config.OutputDirectory == "" {
		summary.warnf("Output directory not configured. Skipping.")
		return false
	}

	return true
}

// parseSimulatorProfiles parses simulator configuration files
func parseSimulatorProfiles(parser common.SimulatorParser, simType common.SimulationType, config *common.Config) (*common.ProfileCollection, error) {
	common.Println("\n========================================")
	common.Printf("=== Processing %s ===\n", parser.GetName())
	common.Println("========================================")

	configPath := common.ConfiguredSimulatorPath(config, simType)
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
	common.Printf("\n%s: Found %d profiles: %s\n", parser.GetName(), len(profileNames), strings.Join(profileNames, ", "))
}

// matchesFilter checks if a name matches the filter (case-insensitive partial match)
func matchesFilter(name, filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(filter))
}

// MatchesModuleFilter matches a DCS module against the filter using both the raw
// profile name ("M-2000C") and the normalized config key ("m2000c"). The GUI
// builds its export targets from the config keys, while the CLI's -f is usually
// typed as part of the profile name; both must select the same module.
func MatchesModuleFilter(moduleName, filter string) bool {
	return matchesFilter(moduleName, filter) ||
		matchesFilter(common.NormalizeModuleName(moduleName), filter)
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
//
// Its parameter is named batch, not ctx: this file also carries a real
// context.Context, and the two sharing a name is what kept cancellation from
// being threaded down here.
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
func runBatchExport(batch *batchExportContext) (int, []*common.ExportDevice) {
	exportCount := 0
	var allExportDevices []*common.ExportDevice

	// Group devices by template
	templateGroups := common.GroupDevicesByTemplate(batch.deviceTemplates)

	for _, deviceGUIDs := range templateGroups {
		template := batch.deviceTemplates[deviceGUIDs[0]]
		if template == nil {
			continue
		}

		exportDevice := createExportDeviceForGroup(deviceGUIDs, template, batch)
		enrich(exportDevice, batch)

		allExportDevices = append(allExportDevices, exportDevice)
		exportCount++
	}

	// Handle devices without templates (create individual export devices)
	for guid, device := range batch.devices {
		if device.IsVirtual {
			continue
		}

		// Skip if device already has a template
		if _, hasTemplate := batch.deviceTemplates[guid]; hasTemplate {
			continue
		}

		exportDevice := createExportDeviceWithoutTemplate(guid, device, batch)
		enrich(exportDevice, batch)

		allExportDevices = append(allExportDevices, exportDevice)
		exportCount++
	}

	return exportCount, allExportDevices
}

// createExportDeviceForGroup creates an ExportDevice for a group of devices sharing a template
func createExportDeviceForGroup(deviceGUIDs []string, template *common.Template, batch *batchExportContext) *common.ExportDevice {
	mergedProfile := createMergedProfile(batch)
	mergeDeviceBindings(deviceGUIDs, batch, mergedProfile)
	representativeDevice := batch.devices[deviceGUIDs[0]]

	return &common.ExportDevice{
		Device:   representativeDevice,
		Template: template,
		Profile:  mergedProfile,
	}
}

// createMergedProfile creates a profile combining bindings from all devices in the group
func createMergedProfile(batch *batchExportContext) *common.Profile {
	// Calculate profile name
	var profileName string
	switch {
	case batch.simType == common.DCSWorld && batch.moduleName != "":
		profileName = fmt.Sprintf("DCS World / %s", strings.ToUpper(batch.moduleName))
	case len(batch.profiles) > 0:
		profileName = batch.profiles[0].Name
	default:
		profileName = "External Bindings"
	}

	mergedProfile := &common.Profile{
		Name:              profileName,
		SimType:           batch.simType,
		Module:            batch.moduleName,
		Devices:           make(map[string]*common.Device),
		Bindings:          make([]common.Binding, 0),
		ModifierDeviceMap: make(map[string]common.ModifierInfo),
	}

	// Copy ModifierDeviceMap from fullProfile if available
	if batch.fullProfile != nil && batch.fullProfile.ModifierDeviceMap != nil {
		for key, value := range batch.fullProfile.ModifierDeviceMap {
			mergedProfile.ModifierDeviceMap[key] = value
		}
	}

	return mergedProfile
}

// appendBindingsFor copies onto the merged profile every binding of every source
// profile that belongs to this device.
//
// The GUID match is partial because the same controller is named with a
// 5-segment GUID by DCS and a 4-segment one by IL-2, and a TARGET profile may
// use either.
func appendBindingsFor(deviceGUID string, batch *batchExportContext, mergedProfile *common.Profile) {
	for _, profile := range batch.profiles {
		for _, binding := range profile.Bindings {
			if common.MatchGUIDPartial(binding.DeviceGUID, deviceGUID) {
				mergedProfile.Bindings = append(mergedProfile.Bindings, binding)
			}
		}
	}
}

// mergeDeviceBindings registers each device of the group on the merged profile and
// copies over the bindings that belong to it.
func mergeDeviceBindings(deviceGUIDs []string, batch *batchExportContext, mergedProfile *common.Profile) {
	for _, deviceGUID := range deviceGUIDs {
		device := batch.devices[deviceGUID]
		if device == nil {
			continue
		}

		mergedProfile.Devices[deviceGUID] = device
		appendBindingsFor(deviceGUID, batch, mergedProfile)
	}

	// If no bindings found, create empty profile for external bindings
	if len(mergedProfile.Bindings) == 0 && batch.moduleName == "" {
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
		common.Printf("  → %s\n", name)
	}
}

// createExportDeviceWithoutTemplate creates an ExportDevice without template
func createExportDeviceWithoutTemplate(deviceGUID string, device *common.Device, batch *batchExportContext) *common.ExportDevice {
	mergedProfile := createMergedProfile(batch)
	appendBindingsFor(deviceGUID, batch, mergedProfile)
	mergedProfile.Devices[deviceGUID] = device

	return &common.ExportDevice{
		Device:   device,
		Template: nil,
		Profile:  mergedProfile,
	}
}

// enrich applies enrichment functions to add bindings from external tools
func enrich(exportDevice *common.ExportDevice, batch *batchExportContext) {
	// Add enrichment bindings (Gremlins, TARGET, OpenKneeboard, SRS, etc.)
	for _, enrichFunc := range batch.enrichmentFuncs {
		enrichFunc(exportDevice, batch.fullProfile, batch.config)
	}
}

// finishBatchExport exports CSV and optionally generates SVG from CSV.
func finishBatchExport(ctx context.Context, allExportDevices []*common.ExportDevice, config *common.Config, noSVG bool, summary *ExportSummary) {
	if len(allExportDevices) == 0 {
		return
	}
	// Assign modifier numbers to all bindings before export
	AssignModifierNumbers(allExportDevices)

	csvPath := filepath.Join(config.OutputDirectory, "export.csv")
	if err := csv.ExportToCSV(allExportDevices, csvPath, config); err != nil {
		summary.errorf("CSV export failed: %v", err)
		return
	}

	summary.CSVPath = csvPath
	common.Printf("\n✓ CSV exported: %s\n", csvPath)

	// Generate SVG/PNG from CSV (skip if --no-svg flag is set)
	if noSVG {
		return
	}

	validationErrors, err := svg.GenerateSVGFromCSV(ctx, csvPath, config)
	summary.ValidationErrors = validationErrors

	// A cancelled run is not a failed one. The interface already suppresses the
	// returned error when the user pressed Cancel, but the summary kept
	// "context canceled" in Errors and the Generate tab rendered it in red.
	if err != nil && ctx.Err() == nil {
		summary.errorf("SVG generation from CSV failed: %v", err)
	}
}

// exportModule exports diagrams for a set of profiles (used by both DCS and IL-2)
func exportModule(profiles []*common.Profile, simType common.SimulationType, moduleName string, config *common.Config, enrichmentFuncs []EnrichmentFunc) (int, []*common.ExportDevice) {
	// Get devices
	devices := collectDevicesFromProfiles(profiles)

	// Load template paths only (no SVG content)
	deviceTemplates := loadDeviceTemplatePaths(devices, config)

	// Build export context
	batch := buildExportContext(profiles, simType, moduleName, config, devices, deviceTemplates, enrichmentFuncs)

	return runBatchExport(batch)
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
func exportDCSBatch(ctx context.Context, profiles *common.ProfileCollection, config *common.Config, enrichmentFuncs []EnrichmentFunc, filter string) *exportResult {
	systemProfiles, moduleProfiles := separateDCSProfiles(profiles)

	if len(moduleProfiles) == 0 {
		common.Println("⚠ No module profiles found. Export cancelled.")
		return &exportResult{}
	}

	result := exportDCSModules(ctx, moduleProfiles, systemProfiles, config, enrichmentFuncs, filter)

	common.Printf("\n=== Total: %d device(s) processed ===\n", result.totalExported)

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
func exportDCSModules(ctx context.Context, moduleProfiles map[string][]*common.Profile, systemProfiles []*common.Profile, config *common.Config, enrichmentFuncs []EnrichmentFunc, filter string) *dcsExportResult {
	result := &dcsExportResult{
		allExportDevices: make([]*common.ExportDevice, 0),
	}

	moduleNames := slices.Sorted(maps.Keys(moduleProfiles))

	matched := 0
	for _, moduleName := range moduleNames {
		// One check per module: a rig with twenty aircraft used to run every one
		// of them before noticing the user had pressed Cancel.
		if ctx.Err() != nil {
			return result
		}

		if filter != "" && !MatchesModuleFilter(moduleName, filter) {
			continue
		}
		matched++

		exportDCSModule(moduleName, moduleProfiles[moduleName], systemProfiles, config, enrichmentFuncs, result)
	}

	if matched == 0 && filter != "" {
		common.Printf("\n⚠ No DCS module matches filter %q. Available: %s\n",
			filter, strings.Join(moduleNames, ", "))
	}

	return result
}

// exportDCSModule exports a single DCS module
func exportDCSModule(moduleName string, moduleProfilesList []*common.Profile, systemProfiles []*common.Profile, config *common.Config, enrichmentFuncs []EnrichmentFunc, result *dcsExportResult) {
	common.Printf("\n=== Module: %s ===\n", moduleName)

	// Combine module and system profiles
	allProfiles := make([]*common.Profile, 0, len(moduleProfilesList)+len(systemProfiles))
	allProfiles = append(allProfiles, moduleProfilesList...)
	allProfiles = append(allProfiles, systemProfiles...)

	// Show "Collecting bindings" section
	common.Printf("\n=== Collecting bindings ===\n")
	common.Println()
	displayDeviceListCSVMode(allProfiles)
	common.Println()

	// Export this module
	moduleExported, exportDevices := exportModule(allProfiles, common.DCSWorld, moduleName, config, enrichmentFuncs)
	result.allExportDevices = append(result.allExportDevices, exportDevices...)

	common.Printf("✓ %d device(s) processed\n", moduleExported)

	result.totalExported += moduleExported
}

// exportNonModularBatch handles batch export for non-modular simulators (IL-2)
func exportNonModularBatch(profiles *common.ProfileCollection, config *common.Config, simType common.SimulationType, enrichmentFuncs []EnrichmentFunc) *exportResult {
	devices := common.GetAllDevicesFromProfiles(profiles)
	if len(devices) == 0 {
		common.Println("No device detected.")
		return &exportResult{}
	}

	// Show "Collecting bindings" section
	common.Println("\n=== Collecting bindings ===")
	common.Println()
	displayDeviceListCSVMode(profiles.Profiles)
	common.Println()

	// Export
	exportCount, allExportDevices := exportModule(profiles.Profiles, simType, "", config, enrichmentFuncs)

	common.Printf("✓ %d device(s) processed\n", exportCount)

	return &exportResult{
		devices: allExportDevices,
	}
}
