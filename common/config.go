package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// ConfigFileName is the name of the configuration file
var ConfigFileName = "mapping_config.yaml"

// SetConfigFileName allows setting a custom configuration filename
func SetConfigFileName(filename string) {
	ConfigFileName = filename
}

// GetConfigFileName returns the current config filename
func GetConfigFileName() string {
	return ConfigFileName
}

// MakeRelativePath converts an absolute template path to a path relative to templatesDir.
// If the path is already relative or can't be made relative, returns the original path.
func MakeRelativePath(absolutePath, templatesDir string) string {
	if absolutePath == "" || templatesDir == "" {
		return absolutePath
	}

	// Clean both paths
	cleanAbsolute := filepath.Clean(absolutePath)
	cleanTemplates := filepath.Clean(templatesDir)

	// Try to make relative
	relPath, err := filepath.Rel(cleanTemplates, cleanAbsolute)
	if err != nil {
		// If can't make relative, return original
		return absolutePath
	}

	// If relative path starts with "..", the file is not under templates directory
	// In this case, keep the absolute path
	if strings.HasPrefix(relPath, "..") {
		return absolutePath
	}

	return relPath
}

// MakeAbsolutePath reconstructs an absolute path from a relative path and templates directory
// If the path is already absolute, returns it as-is
func MakeAbsolutePath(relativePath, templatesDir string) string {
	if relativePath == "" {
		return ""
	}

	// If already absolute, return as-is
	if filepath.IsAbs(relativePath) {
		return relativePath
	}

	// Combine templates directory with relative path
	return filepath.Join(templatesDir, relativePath)
}

// DeviceTemplateMapping represents the association of a device with a template
type DeviceTemplateMapping struct {
	DeviceGUID         string   `yaml:"device_guid"`
	AlternateGUIDs     []string `yaml:"alternate_guids,omitempty"` // Additional GUIDs for the same device (e.g., IL-2 vs DCS format)
	DeviceName         string   `yaml:"device_name"`
	TemplateFilepath   string   `yaml:"template_filepath"`
	SkipTemplate       bool     `yaml:"skip_template,omitempty"`        // User chose not to associate a template
	DeviceTargetNumber int      `yaml:"device_target_number,omitempty"` // TARGET device number (1001, 1002, etc.) if using TARGET
}

// ModuleConfig represents configuration for a specific module (e.g., M-2000C, F/A-18C)
type ModuleConfig struct {
	GremlinsProfileFilepath string `yaml:"gremlins_profile_filepath,omitempty"`
	TargetProfileFilepath   string `yaml:"target_profile_filepath,omitempty"`
}

// SimulatorConfig represents configuration for a specific simulator
type SimulatorConfig struct {
	// For DCS World: modules (M-2000C, FA-18C, etc.)
	Modules map[string]*ModuleConfig `yaml:"modules,omitempty"`

	// Simulator-specific paths
	DCSPath      string `yaml:"dcs_path,omitempty"`       // DCS World installation path (to find Config/Input)
	IL2InputPath string `yaml:"il2_input_path,omitempty"` // IL-2 input configuration path
	SRSPath      string `yaml:"srs_path,omitempty"`       // SRS configuration path

	// For IL-2 and other non-modular sims: direct config
	GremlinsProfileFilepath string `yaml:"gremlins_profile_filepath,omitempty"`
	TargetProfileFilepath   string `yaml:"target_profile_filepath,omitempty"`
}

// Config represents the saved configuration organized by simulator
type Config struct {
	TemplatesDirectory            string                      `yaml:"templates_directory,omitempty"` // Global templates directory
	OutputDirectory               string                      `yaml:"output_directory,omitempty"`    // Global output directory (default: ./output)
	DeviceMappings                []DeviceTemplateMapping     `yaml:"device_mappings,omitempty"`     // Global device to template mappings
	Simulators                    map[string]*SimulatorConfig `yaml:"simulators"`
	DrawIOPath                    string                      `yaml:"drawio_path,omitempty"`
	OpenKneeboardProfilesFilepath string                      `yaml:"openkneeboard_profiles_filepath,omitempty"`
	KeyboardLayout                string                      `yaml:"keyboard_layout,omitempty"` // "qwerty" or "azerty"
}

// NormalizeModuleName normalizes module name from profile name
// E.g., "M-2000C" -> "m2000c", "FA-18C_hornet" -> "fa18c", "P-47D-30" -> "p47d"
func NormalizeModuleName(profileName string) string {
	if profileName == "" {
		return ""
	}

	// Common mappings for DCS modules
	mappings := map[string]string{
		"M-2000C":       "m2000c",
		"FA-18C_hornet": "fa18c",
		"F-16C_50":      "f16c",
		"P-47D-30":      "p47d",
		"F-15E":         "f15e",
		"AH-64D_BLK_II": "ah64d",
		"Ka-50":         "ka50",
		"Mi-8MT":        "mi8",
		"UH-1H":         "uh1h",
		"A-10C":         "a10c",
		"A-10C_2":       "a10c2",
	}

	if normalized, ok := mappings[profileName]; ok {
		return normalized
	}

	// Generic normalization: keep ASCII letters and digits, lowercased
	var result strings.Builder
	result.Grow(len(profileName))
	for _, c := range profileName {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			result.WriteRune(c)
		case c >= 'A' && c <= 'Z':
			result.WriteRune(unicode.ToLower(c))
		}
	}

	return result.String()
}

// getCommonProfilePath is a generic helper to find a common profile path across all modules/simulators
func (c *Config) getCommonProfilePath(getSimPath func(*SimulatorConfig) string, getModulePath func(*ModuleConfig) string) string {
	if c.Simulators == nil {
		return ""
	}

	var commonPath string
	firstFound := false

	// Check all simulators
	for _, simConfig := range c.Simulators {
		if simConfig == nil {
			continue
		}

		// Check modules (for DCS)
		if simConfig.Modules != nil {
			for _, moduleConfig := range simConfig.Modules {
				path := getModulePath(moduleConfig)
				if path != "" {
					if !firstFound {
						commonPath = path
						firstFound = true
					} else if commonPath != path {
						return "" // Different paths found
					}
				}
			}
		}

		// Check flat simulator config (for IL-2, etc.)
		path := getSimPath(simConfig)
		if path != "" {
			if !firstFound {
				commonPath = path
				firstFound = true
			} else if commonPath != path {
				return "" // Different paths found
			}
		}
	}

	return commonPath
}

// GetCommonGremlinsProfilePath returns a common Gremlins profile path if all configured modules/simulators use the same one
// Returns empty string if no common path exists or if configurations differ
func (c *Config) GetCommonGremlinsProfilePath() string {
	return c.getCommonProfilePath(
		func(sc *SimulatorConfig) string { return sc.GremlinsProfileFilepath },
		func(mc *ModuleConfig) string { return mc.GremlinsProfileFilepath },
	)
}

// GetCommonTargetProfilePath returns a common TARGET profile path if all configured modules/simulators use the same one
// Returns empty string if no common path exists or if configurations differ
func (c *Config) GetCommonTargetProfilePath() string {
	return c.getCommonProfilePath(
		func(sc *SimulatorConfig) string { return sc.TargetProfileFilepath },
		func(mc *ModuleConfig) string { return mc.TargetProfileFilepath },
	)
}

// GetTemplateFilepathForDevice searches for an existing template_filepath for a device GUID
// Returns the relative template filepath and true if found, empty string and false otherwise
// Note: GUID comparison is case-insensitive and compares the full GUID
func (c *Config) GetTemplateFilepathForDevice(deviceGUID string) (string, bool) {
	// Normalize the search GUID for comparison (case-insensitive, full GUID)
	normalizedSearchGUID := NormalizeGUID(deviceGUID)

	// Search in global device mappings
	for _, mapping := range c.DeviceMappings {
		if NormalizeGUID(mapping.DeviceGUID) == normalizedSearchGUID &&
			mapping.TemplateFilepath != "" && !mapping.SkipTemplate {
			return mapping.TemplateFilepath, true
		}
	}

	return "", false
}

// GetModuleConfig gets or creates config for a specific module (DCS only)
// For non-DCS simulators, this should not be used - use GetSimulatorConfig instead
func (c *Config) GetModuleConfig(simType SimulationType, moduleName string) *ModuleConfig {
	if c.Simulators == nil {
		c.Simulators = make(map[string]*SimulatorConfig)
	}

	simKey := simType.GetConfigKey()
	if c.Simulators[simKey] == nil {
		c.Simulators[simKey] = &SimulatorConfig{}
	}

	simConfig := c.Simulators[simKey]

	// For DCS World, use modules
	if simType == DCSWorld {
		if simConfig.Modules == nil {
			simConfig.Modules = make(map[string]*ModuleConfig)
		}

		moduleKey := NormalizeModuleName(moduleName)
		if simConfig.Modules[moduleKey] == nil {
			simConfig.Modules[moduleKey] = &ModuleConfig{}
		}

		return simConfig.Modules[moduleKey]
	}

	// For other sims, this function should not be called
	// Return nil to indicate error
	return nil
}

// GetSimulatorConfig gets or creates config for a simulator (backward compatibility)
func (c *Config) GetSimulatorConfig(simType SimulationType) *SimulatorConfig {
	if c.Simulators == nil {
		c.Simulators = make(map[string]*SimulatorConfig)
	}

	key := simType.GetConfigKey()
	if c.Simulators[key] == nil {
		newConfig := &SimulatorConfig{}

		// For DCS World, initialize with modules structure
		if simType == DCSWorld {
			newConfig.Modules = make(map[string]*ModuleConfig)
		}
		// Note: DeviceMappings are now at Config level, not SimulatorConfig level

		c.Simulators[key] = newConfig
	}

	return c.Simulators[key]
}

// LoadConfig loads configuration from YAML file
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(ConfigFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				Simulators: make(map[string]*SimulatorConfig),
			}, nil
		}
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}

	// Initialize map if nil
	if config.Simulators == nil {
		config.Simulators = make(map[string]*SimulatorConfig)
	}

	// Initialize DeviceMappings if nil
	if config.DeviceMappings == nil {
		config.DeviceMappings = []DeviceTemplateMapping{}
	}

	return &config, nil
}

// SaveConfig saves configuration to YAML file
func SaveConfig(config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error generating YAML: %w", err)
	}

	if err := os.WriteFile(ConfigFileName, data, 0644); err != nil {
		return fmt.Errorf("error writing config: %w", err)
	}

	return nil
}

// exactPrimaryIndex returns the index of the mapping whose primary GUID equals
// normalizedGUID (already passed through NormalizeGUID), or -1.
func (c *Config) exactPrimaryIndex(normalizedGUID string) int {
	for i := range c.DeviceMappings {
		if NormalizeGUID(c.DeviceMappings[i].DeviceGUID) == normalizedGUID {
			return i
		}
	}
	return -1
}

// exactAlternateIndex returns the index of the mapping carrying an alternate GUID
// equal to normalizedGUID (already passed through NormalizeGUID), or -1.
func (c *Config) exactAlternateIndex(normalizedGUID string) int {
	for i := range c.DeviceMappings {
		for _, altGUID := range c.DeviceMappings[i].AlternateGUIDs {
			if NormalizeGUID(altGUID) == normalizedGUID {
				return i
			}
		}
	}
	return -1
}

// partialIndex returns the index of the first mapping whose primary or alternate
// GUID partially matches deviceGUID (cross-simulator GUID formats), or -1.
func (c *Config) partialIndex(deviceGUID string) int {
	for i := range c.DeviceMappings {
		if MatchGUIDPartial(c.DeviceMappings[i].DeviceGUID, deviceGUID) {
			return i
		}
		for _, altGUID := range c.DeviceMappings[i].AlternateGUIDs {
			if MatchGUIDPartial(altGUID, deviceGUID) {
				return i
			}
		}
	}
	return -1
}

// exactMappingIndex finds a mapping matching deviceGUID exactly, looking at the
// primary GUID before the alternates. Returns -1 when there is no exact match.
// Partial matches are deliberately excluded: several distinct physical devices can
// share the same first three GUID segments, so writing to a partial match would
// overwrite an unrelated device's entry.
func (c *Config) exactMappingIndex(deviceGUID string) int {
	normalizedGUID := NormalizeGUID(deviceGUID)
	if i := c.exactPrimaryIndex(normalizedGUID); i >= 0 {
		return i
	}
	return c.exactAlternateIndex(normalizedGUID)
}

// GetTemplateMappingForDevice retrieves the template associated with a device from global config.
// Priority: 1) Exact match on alternate GUIDs, 2) Exact match on primary GUID, 3) Partial match.
// Alternates are checked first so that a device explicitly registered under a second
// GUID wins over another device sharing its first three GUID segments.
func (c *Config) GetTemplateMappingForDevice(deviceGUID string) *DeviceTemplateMapping {
	normalizedGUID := NormalizeGUID(deviceGUID)

	if i := c.exactAlternateIndex(normalizedGUID); i >= 0 {
		return &c.DeviceMappings[i]
	}
	if i := c.exactPrimaryIndex(normalizedGUID); i >= 0 {
		return &c.DeviceMappings[i]
	}
	if i := c.partialIndex(deviceGUID); i >= 0 {
		return &c.DeviceMappings[i]
	}
	return nil
}

// UpdateDeviceMapping updates or adds a device → template mapping in global config.
// An existing mapping is reused only on an exact GUID match (primary or alternate);
// otherwise a new mapping is appended.
func (c *Config) UpdateDeviceMapping(deviceGUID, deviceName, templatePath, templatesDir string) {
	relativePath := MakeRelativePath(templatePath, templatesDir)

	if i := c.exactMappingIndex(deviceGUID); i >= 0 {
		c.DeviceMappings[i].TemplateFilepath = relativePath
		c.DeviceMappings[i].DeviceName = deviceName
		c.DeviceMappings[i].SkipTemplate = false
		return
	}

	c.DeviceMappings = append(c.DeviceMappings, DeviceTemplateMapping{
		DeviceGUID:       deviceGUID,
		DeviceName:       deviceName,
		TemplateFilepath: relativePath,
	})
}

// UpdateDeviceTargetNumber updates the DeviceTargetNumber for a device in a slice of DeviceMappings
// This is a helper function used when applying TARGET device mappings
func UpdateDeviceTargetNumber(mappings []DeviceTemplateMapping, deviceGUID string, targetNumber int) {
	for i := range mappings {
		if MatchGUIDPartial(mappings[i].DeviceGUID, deviceGUID) {
			mappings[i].DeviceTargetNumber = targetNumber
			return
		}
		// Check alternate GUIDs
		for _, altGUID := range mappings[i].AlternateGUIDs {
			if MatchGUIDPartial(altGUID, deviceGUID) {
				mappings[i].DeviceTargetNumber = targetNumber
				return
			}
		}
	}
}

// MarkDeviceAsSkipped marks a device as explicitly not having a template in global config.
// An existing mapping is reused only on an exact GUID match (primary or alternate);
// otherwise a new mapping carrying the skip flag is appended.
func (c *Config) MarkDeviceAsSkipped(deviceGUID, deviceName string) {
	if i := c.exactMappingIndex(deviceGUID); i >= 0 {
		c.DeviceMappings[i].SkipTemplate = true
		c.DeviceMappings[i].TemplateFilepath = ""
		c.DeviceMappings[i].DeviceName = deviceName
		return
	}

	c.DeviceMappings = append(c.DeviceMappings, DeviceTemplateMapping{
		DeviceGUID:       deviceGUID,
		DeviceName:       deviceName,
		TemplateFilepath: "",
		SkipTemplate:     true,
	})
}

// GetTemplatePathIfExists returns the template path if it exists and is valid
func (c *Config) GetTemplatePathIfExists(deviceGUID string) (string, bool) {
	mapping := c.GetTemplateMappingForDevice(deviceGUID)
	if mapping == nil {
		return "", false
	}

	// If user explicitly chose to skip template, return empty
	if mapping.SkipTemplate {
		return "", false
	}

	// Get global templates directory
	templatesDir := c.TemplatesDirectory

	// Reconstruct absolute path from templates directory + relative path
	absolutePath := MakeAbsolutePath(mapping.TemplateFilepath, templatesDir)

	// Check that file still exists
	if _, err := os.Stat(absolutePath); err != nil {
		return "", false
	}

	return absolutePath, true
}

// VerifyDrawIOPath checks if draw.io is available at configured or default paths
func VerifyDrawIOPath(config *Config) (string, bool) {
	// First, try configured path from config
	if config != nil && config.DrawIOPath != "" {
		if _, err := os.Stat(config.DrawIOPath); err == nil {
			return config.DrawIOPath, true
		}
	}

	// If not found in config, try default paths
	drawIOPaths := []string{
		"draw.io", // In PATH
		"C:\\Program Files\\draw.io\\draw.io.exe", // Default Windows installation
	}

	for _, path := range drawIOPaths {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}

	return "", false
}

// VerifySRSPaths checks if SRS paths are available at configured or default paths
func VerifySRSPaths(config *Config) (il2Path string, dcsPath string, il2Found bool, dcsFound bool) {
	// Check IL-2 SRS path from simulator config
	if config != nil {
		if il2Config := config.GetSimulatorConfig(IL2Sturmovik); il2Config != nil && il2Config.SRSPath != "" {
			if _, err := os.Stat(il2Config.SRSPath); err == nil {
				il2Path = il2Config.SRSPath
				il2Found = true
			}
		}
	}

	// Try default IL-2 path if not found
	if !il2Found {
		defaultIL2Path := "C:\\Program Files\\IL2-SimpleRadio-Standalone"
		if _, err := os.Stat(defaultIL2Path); err == nil {
			il2Path = defaultIL2Path
			il2Found = true
		}
	}

	// Check DCS SRS path from simulator config
	if config != nil {
		if dcsConfig := config.GetSimulatorConfig(DCSWorld); dcsConfig != nil && dcsConfig.SRSPath != "" {
			if _, err := os.Stat(dcsConfig.SRSPath); err == nil {
				dcsPath = dcsConfig.SRSPath
				dcsFound = true
			}
		}
	}

	// Try default DCS path if not found
	if !dcsFound {
		defaultDCSPath := "C:\\Program Files\\DCS-SimpleRadio-Standalone"
		if _, err := os.Stat(defaultDCSPath); err == nil {
			dcsPath = defaultDCSPath
			dcsFound = true
		}
	}

	return il2Path, dcsPath, il2Found, dcsFound
}

// GetConfiguredDCSModules returns a list of all configured DCS modules in the config
func (c *Config) GetConfiguredDCSModules() []string {
	if c.Simulators == nil {
		return nil
	}

	dcsConfig := c.Simulators["dcs_world"]
	if dcsConfig == nil || dcsConfig.Modules == nil {
		return nil
	}

	modules := make([]string, 0, len(dcsConfig.Modules))
	for moduleName := range dcsConfig.Modules {
		modules = append(modules, moduleName)
	}

	return modules
}

// DuplicateModuleConfig creates a copy of a module configuration for another module
func (c *Config) DuplicateModuleConfig(sourceModule, targetModule string) error {
	if c.Simulators == nil {
		c.Simulators = make(map[string]*SimulatorConfig)
	}

	dcsConfig := c.Simulators["dcs_world"]
	if dcsConfig == nil {
		return fmt.Errorf("no DCS World configuration found")
	}

	if dcsConfig.Modules == nil {
		return fmt.Errorf("no modules configured")
	}

	// Normalize module names for config keys
	normalizedSource := NormalizeModuleName(sourceModule)
	normalizedTarget := NormalizeModuleName(targetModule)

	sourceConfig, exists := dcsConfig.Modules[normalizedSource]
	if !exists {
		return fmt.Errorf("source module %s not found", sourceModule)
	}

	// Create a copy of the source configuration
	// Note: DeviceMappings and TemplatesDirectory are now global and not copied here
	targetConfig := &ModuleConfig{
		GremlinsProfileFilepath: sourceConfig.GremlinsProfileFilepath,
		TargetProfileFilepath:   sourceConfig.TargetProfileFilepath,
	}

	// Set the new module configuration with normalized name
	dcsConfig.Modules[normalizedTarget] = targetConfig

	return nil
}
