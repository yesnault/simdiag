package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"simdiag/common"
)

// ConfigureWorkflowInteractive handles configuration workflow with manual template selection
func ConfigureWorkflowInteractive(profiles *common.ProfileCollection, simType common.SimulationType) {
	config := loadOrCreateConfig()
	if config == nil {
		return
	}

	selectedModule, shouldExit := selectDCSModule(config, profiles, simType)
	if shouldExit {
		return
	}

	templatesDir := configureTemplatesDirectory(config)
	if templatesDir == "" {
		return
	}

	ctx := &exportContext{
		config:         config,
		simType:        simType,
		selectedModule: selectedModule,
	}

	devices := getAndValidateDevices(profiles)
	if devices == nil {
		return
	}

	targetDeviceMappings := configureExternalTools(ctx, devices)

	deviceTemplates := configureDeviceTemplates(ctx, devices, templatesDir)

	updateTargetDeviceMappings(config, targetDeviceMappings)

	saveConfigAfterTemplateSelection(config, ctx, deviceTemplates)
}

// loadOrCreateConfig loads existing config or creates a new one
func loadOrCreateConfig() *common.Config {
	config, err := common.LoadConfig()
	if err != nil {
		fmt.Printf("Warning: unable to load config: %v\n", err)
		config = &common.Config{Simulators: make(map[string]*common.SimulatorConfig)}
	}
	return config
}

// selectDCSModule handles DCS module selection and duplication logic
func selectDCSModule(config *common.Config, profiles *common.ProfileCollection, simType common.SimulationType) (string, bool) {
	if simType != common.DCSWorld {
		return "", false
	}

	modules := common.GetModulesFromProfiles(profiles)
	if len(modules) == 0 {
		showNoModulesMessage()
		return "", true
	}

	showAvailableModules(modules)

	if shouldDuplicateConfig(config, modules) {
		return "", true
	}

	selectedModule := selectModule(modules)
	if selectedModule == "" {
		return "", true
	}

	fmt.Printf("\n=== Configuring module: %s ===\n", selectedModule)
	return selectedModule, false
}

// showNoModulesMessage displays message when no aircraft modules are found
func showNoModulesMessage() {
	fmt.Println("\n⚠ No aircraft module profiles detected.")
	fmt.Println("Only system profiles (Default, UiLayer, CommandMenu) were found.")
	fmt.Println("\nTo configure devices for a specific aircraft:")
	fmt.Println("1. Start DCS World")
	fmt.Println("2. Configure controls for at least one aircraft module")
	fmt.Println("3. Save and exit DCS")
	fmt.Println("4. Run this program again")
	fmt.Println("\nExport cancelled.")
}

// showAvailableModules displays the list of available modules
func showAvailableModules(modules []string) {
	fmt.Printf("\n%d aircraft module(s) found: ", len(modules))
	for i, mod := range modules {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(mod)
	}
	fmt.Println()
}

// shouldDuplicateConfig checks if config duplication is needed and handles it
func shouldDuplicateConfig(config *common.Config, modules []string) bool {
	configuredModules := config.GetConfiguredDCSModules()
	if len(configuredModules) != 1 || len(modules) <= 1 {
		return false
	}

	configuredModule := configuredModules[0]
	fmt.Printf("\n💡 You have configured %s, but %d other module(s) are available.\n", configuredModule, len(modules)-1)

	if !common.AskYesNo("Would you like to apply this configuration to other modules?") {
		return false
	}

	selectedModules := common.SelectMultipleModules(modules, configuredModule)
	if len(selectedModules) == 0 {
		return false
	}

	duplicateModuleConfigs(config, configuredModule, selectedModules)
	saveConfigAndPromptRestart(config)
	return true
}

// duplicateModuleConfigs applies configuration from one module to others
func duplicateModuleConfigs(config *common.Config, sourceModule string, targetModules []string) {
	fmt.Printf("\nApplying configuration from %s to:\n", sourceModule)
	for _, targetModule := range targetModules {
		fmt.Printf("  - %s\n", targetModule)

		err := config.DuplicateModuleConfig(sourceModule, targetModule)
		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
		} else {
			fmt.Printf("    ✓ Configuration applied\n")
		}
	}
}

// saveConfigAndPromptRestart saves config and prompts user to restart
func saveConfigAndPromptRestart(config *common.Config) {
	if err := common.SaveConfig(config); err != nil {
		fmt.Printf("\n⚠ Unable to save configuration: %v\n", err)
		fmt.Println("\nPlease restart the program to use the new configuration.")
		return
	}

	fmt.Println("\n✓ Configuration saved successfully!")
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("⚠  Please restart the program to use the new configuration.")
	fmt.Println(strings.Repeat("=", 80))
}

// selectModule prompts user to select a module
func selectModule(modules []string) string {
	fmt.Println("\n=== Select DCS module to configure ===")
	selectedModule := common.SelectModule(modules)
	if selectedModule == "" {
		fmt.Println("No module selected. Export cancelled.")
		return ""
	}

	return common.NormalizeModuleName(selectedModule)
}

// configureTemplatesDirectory configures and validates templates directory
func configureTemplatesDirectory(config *common.Config) string {
	templatesDir := common.AskTemplatesDirectory(config)
	if templatesDir == "" {
		fmt.Println("Templates directory required. Export cancelled.")
		return ""
	}

	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		fmt.Printf("❌ Directory %s does not exist.\n", templatesDir)
		return ""
	}

	config.TemplatesDirectory = templatesDir

	if config.OutputDirectory == "" {
		config.OutputDirectory = "./output"
	}

	return templatesDir
}

// getAndValidateDevices gets devices from profiles and validates
func getAndValidateDevices(profiles *common.ProfileCollection) []*common.Device {
	devices := common.GetAllDevicesFromProfiles(profiles)

	if len(devices) == 0 {
		fmt.Println("No device detected.")
		return nil
	}

	fmt.Printf("\n%d device(s) detected\n", len(devices))
	return devices
}

// configureExternalTools configures Gremlins and TARGET
func configureExternalTools(ctx *exportContext, devices []*common.Device) []common.TargetDeviceMapping {
	// Ensure module config exists for DCS
	if ctx.isDCSModule() {
		_ = ctx.config.GetModuleConfig(ctx.simType, ctx.selectedModule)
	}

	configureGremlins(ctx)
	return configureTARGET(ctx, devices)
}

// configureGremlins configures Gremlins profile path
func configureGremlins(ctx *exportContext) {
	gremlinsProfilePath := common.AskGremlinsProfilePath(ctx.config)
	if gremlinsProfilePath == "" {
		return
	}

	if ctx.isDCSModule() {
		moduleConfig := ctx.config.GetModuleConfig(ctx.simType, ctx.selectedModule)
		moduleConfig.GremlinsProfileFilepath = gremlinsProfilePath
	} else {
		simConfig := ctx.config.GetSimulatorConfig(ctx.simType)
		simConfig.GremlinsProfileFilepath = gremlinsProfilePath
	}
}

// configureTARGET configures TARGET profile path and returns device mappings
func configureTARGET(ctx *exportContext, devices []*common.Device) []common.TargetDeviceMapping {
	targetProfilePath, targetDeviceMappings := common.AskTargetProfilePath(ctx.config, devices)
	if targetProfilePath == "" {
		return nil
	}

	if ctx.isDCSModule() {
		moduleConfig := ctx.config.GetModuleConfig(ctx.simType, ctx.selectedModule)
		moduleConfig.TargetProfileFilepath = targetProfilePath
	} else {
		simConfig := ctx.config.GetSimulatorConfig(ctx.simType)
		simConfig.TargetProfileFilepath = targetProfilePath
	}

	return targetDeviceMappings
}

// configureDeviceTemplates configures templates for all devices
func configureDeviceTemplates(ctx *exportContext, devices []*common.Device, templatesDir string) map[string]*common.Template {
	useExistingConfig := askUseExistingConfig(ctx)
	deviceTemplates := make(map[string]*common.Template)

	for _, device := range devices {
		template := configureDeviceTemplate(ctx, device, templatesDir, &useExistingConfig)
		if template != nil {
			deviceTemplates[device.GUID] = template
		}
	}

	return deviceTemplates
}

// askUseExistingConfig asks if user wants to use existing configuration
func askUseExistingConfig(ctx *exportContext) bool {
	existingMappings := ctx.getExistingMappings()
	if len(existingMappings) == 0 {
		return false
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("An existing configuration was found.")
	useExisting := common.AskYesNo("Would you like to use the existing configuration for all devices?")
	fmt.Println(strings.Repeat("=", 80))

	return useExisting
}

// configureDeviceTemplate configures template for a single device
func configureDeviceTemplate(ctx *exportContext, device *common.Device, templatesDir string, useExistingConfig *bool) *common.Template {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("Device: %s\n", device.Name)
	fmt.Printf("GUID: %s\n", device.GUID)

	if device.IsVirtual {
		fmt.Println("⊘ Virtual device (no template needed)")
		return nil
	}

	// Check if device is explicitly skipped
	mapping := ctx.getTemplateMapping(device.GUID)
	if mapping != nil && mapping.SkipTemplate {
		fmt.Println("⊊ Device ignored")
		return nil
	}

	// Try to load existing template if using existing config
	if *useExistingConfig {
		template := tryLoadExistingTemplate(ctx, device.GUID, useExistingConfig)
		if template != nil {
			return template
		}
	}

	// Try interactive configuration
	return configureDeviceTemplateInteractive(ctx, device, templatesDir)
}

// tryLoadExistingTemplate attempts to load template from existing config
func tryLoadExistingTemplate(ctx *exportContext, deviceGUID string, useExistingConfig *bool) *common.Template {
	existingPath, exists := ctx.getTemplatePathIfExists(deviceGUID)

	if !exists {
		fmt.Println("⚠ No template configured for this device")
		return nil
	}

	template, err := common.LoadTemplate(existingPath)
	if err != nil {
		fmt.Printf("❌ Error loading template: %v\n", err)
		*useExistingConfig = false
		return nil
	}

	fmt.Printf("✓ Template loaded: %s\n", template.Name)
	return template
}

// configureDeviceTemplateInteractive handles interactive template configuration
func configureDeviceTemplateInteractive(ctx *exportContext, device *common.Device, templatesDir string) *common.Template {
	// Try loading template from current module
	template := tryLoadTemplateFromCurrentModule(ctx, device.GUID)
	if template != nil {
		return template
	}

	// Try loading template from other modules
	template = tryLoadTemplateFromOtherModule(ctx, device, templatesDir)
	if template != nil {
		return template
	}

	// Ask if user wants to associate a template
	return selectNewTemplate(ctx, device, templatesDir)
}

// tryLoadTemplateFromCurrentModule tries to load template from current module config
func tryLoadTemplateFromCurrentModule(ctx *exportContext, deviceGUID string) *common.Template {
	existingPath, exists := ctx.getTemplatePathIfExists(deviceGUID)
	if !exists {
		return nil
	}

	relPath, _ := filepath.Rel(".", existingPath)
	if !common.AskYesNo(fmt.Sprintf("Use previous template (%s)?", relPath)) {
		return nil
	}

	template, err := common.LoadTemplate(existingPath)
	if err != nil {
		fmt.Printf("❌ Error loading template: %v\n", err)
		return nil
	}

	fmt.Printf("✓ Template loaded: %s\n", template.Name)
	return template
}

// tryLoadTemplateFromOtherModule tries to load template from other modules
func tryLoadTemplateFromOtherModule(ctx *exportContext, device *common.Device, templatesDir string) *common.Template {
	existingTemplateFilepath, foundInOther := ctx.config.GetTemplateFilepathForDevice(device.GUID)
	if !foundInOther {
		return nil
	}

	if !common.AskYesNo(fmt.Sprintf("A template is already configured for this device (%s). Use it?", existingTemplateFilepath)) {
		return nil
	}

	fullPath := filepath.Join(templatesDir, existingTemplateFilepath)
	template, err := common.LoadTemplate(fullPath)
	if err != nil {
		fmt.Printf("❌ Error loading template: %v\n", err)
		return nil
	}

	ctx.updateDeviceMapping(device.GUID, device.Name, fullPath)
	fmt.Printf("✓ Template loaded: %s\n", template.Name)
	return template
}

// selectNewTemplate prompts user to select a new template
func selectNewTemplate(ctx *exportContext, device *common.Device, templatesDir string) *common.Template {
	if !common.AskYesNo("Would you like to associate a template with this device?") {
		fmt.Println("⊊ Device ignored")
		ctx.markDeviceAsSkipped(device.GUID, device.Name)
		return nil
	}

	templatePath, err := common.SelectTemplateInteractive(templatesDir, device.Name)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return nil
	}

	if templatePath == "" {
		fmt.Println("⊊ No template selected")
		ctx.markDeviceAsSkipped(device.GUID, device.Name)
		return nil
	}

	template, err := common.LoadTemplate(templatePath)
	if err != nil {
		fmt.Printf("❌ Error loading template: %v\n", err)
		return nil
	}

	ctx.updateDeviceMapping(device.GUID, device.Name, templatePath)
	return template
}

// updateTargetDeviceMappings updates device mappings with TARGET device numbers
func updateTargetDeviceMappings(config *common.Config, targetDeviceMappings []common.TargetDeviceMapping) {
	if len(targetDeviceMappings) == 0 {
		return
	}

	for _, tdm := range targetDeviceMappings {
		common.UpdateDeviceTargetNumber(config.DeviceMappings, tdm.DeviceGUID, tdm.DeviceNumber)
	}
}

// saveConfigAfterTemplateSelection saves config and shows completion summary
func saveConfigAfterTemplateSelection(config *common.Config, ctx *exportContext, deviceTemplates map[string]*common.Template) {
	existingMappings := ctx.getExistingMappings()

	if len(existingMappings) > 0 || len(deviceTemplates) > 0 {
		if err := common.SaveConfig(config); err != nil {
			fmt.Printf("\n⚠ Unable to save config: %v\n", err)
		} else {
			fmt.Printf("\n✓ Configuration saved in %s\n", common.ConfigFileName)
		}
	}

	if len(deviceTemplates) == 0 {
		fmt.Println("\n⚠ No template selected.")
		fmt.Println("\n✓ Configuration saved successfully!")
		return
	}

	if err := common.SaveConfig(config); err != nil {
		fmt.Printf("\n⚠ Unable to save config: %v\n", err)
		return
	}

	showCompletionSummary(config, ctx)
}

// showCompletionSummary displays final configuration summary
func showCompletionSummary(config *common.Config, ctx *exportContext) {
	outputPath := buildOutputPath(config, ctx)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✓ Configuration completed successfully!")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\nConfiguration saved to: %s\n", common.ConfigFileName)
	fmt.Printf("Diagrams will be generated in: %s\n", outputPath)
	fmt.Printf("CSV will be generated in: %s/export.csv\n", config.OutputDirectory)
	fmt.Printf("\nTo generate diagrams, run: ./simdiag.exe -b\n")
	fmt.Println(strings.Repeat("=", 80))
}

// buildOutputPath builds the output path based on simulator type
func buildOutputPath(config *common.Config, ctx *exportContext) string {
	switch {
	case ctx.isDCSModule():
		return filepath.Join(config.OutputDirectory, "dcs-"+ctx.selectedModule)
	case ctx.simType == common.IL2Sturmovik:
		return filepath.Join(config.OutputDirectory, "il2")
	default:
		return config.OutputDirectory
	}
}

// exportContext holds configuration context for exports
type exportContext struct {
	config         *common.Config
	simType        common.SimulationType
	selectedModule string
}

// isDCSModule returns true if this is a DCS module context
func (ctx *exportContext) isDCSModule() bool {
	return ctx.simType == common.DCSWorld && ctx.selectedModule != ""
}

// getTemplatePathIfExists gets template path from config based on context
func (ctx *exportContext) getTemplatePathIfExists(deviceGUID string) (string, bool) {
	if ctx.isDCSModule() {
		return ctx.config.GetTemplatePathIfExistsInModule(deviceGUID)
	}
	return ctx.config.GetTemplatePathIfExists(deviceGUID)
}

// getTemplateMapping gets device template mapping from global config
func (ctx *exportContext) getTemplateMapping(deviceGUID string) *common.DeviceTemplateMapping {
	// Device mappings are now global
	return ctx.config.GetTemplateMappingForDevice(deviceGUID)
}

// markDeviceAsSkipped marks device as skipped in global config
func (ctx *exportContext) markDeviceAsSkipped(deviceGUID, deviceName string) {
	// Device mappings are now global
	ctx.config.MarkDeviceAsSkipped(deviceGUID, deviceName)
}

// updateDeviceMapping updates device mapping in global config
func (ctx *exportContext) updateDeviceMapping(deviceGUID, deviceName, templatePath string) {
	// Get global templates directory
	templatesDir := ctx.config.TemplatesDirectory

	// Device mappings are now global
	ctx.config.UpdateDeviceMapping(deviceGUID, deviceName, templatePath, templatesDir)
}

// getExistingMappings gets existing device mappings from global config
func (ctx *exportContext) getExistingMappings() []common.DeviceTemplateMapping {
	// Device mappings are now global
	return ctx.config.DeviceMappings
}
