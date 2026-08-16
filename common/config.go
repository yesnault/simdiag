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

// SimulatorConfig represents configuration for a specific simulator.
//
// DCS aircraft are deliberately absent. They used to be declared here, one entry
// per module, each able to carry its own Gremlins and TARGET profile. A
// capability no configuration ever used, since a pilot runs one Gremlins profile
// for the whole game. Modules are now detected from DCSPath (dcs.ListModules) and
// the tool paths below apply to all of them.
type SimulatorConfig struct {
	// Simulator-specific paths
	DCSPath      string `yaml:"dcs_path,omitempty"`       // DCS World installation path (to find Config/Input)
	IL2InputPath string `yaml:"il2_input_path,omitempty"` // IL-2 input configuration path

	// External tool profiles, for every module of this simulator
	GremlinsProfileFilepath string `yaml:"gremlins_profile_filepath,omitempty"`
	TargetProfileFilepath   string `yaml:"target_profile_filepath,omitempty"`
}

// SimulatorIsConfigured reports whether a simulator has the one setting it cannot
// be exported without. Both the batch validation in cmd/simdiag and the
// per-simulator gate in workflow ask this, and they used to answer differently:
// the former never checked DCSPath, so a DCS-only configuration was rejected
// outright.
func SimulatorIsConfigured(simType SimulationType, simConfig *SimulatorConfig) bool {
	if simConfig == nil {
		return false
	}
	if simType == DCSWorld {
		return simConfig.DCSPath != ""
	}
	return simConfig.IL2InputPath != ""
}

// RequiredPathSetting names the config key SimulatorIsConfigured tests, so a
// message about a missing one can name it.
func RequiredPathSetting(simType SimulationType) string {
	if simType == DCSWorld {
		return "dcs_path"
	}
	return "il2_input_path"
}

// Config represents the saved configuration organized by simulator
type Config struct {
	TemplatesDirectory            string                      `yaml:"templates_directory,omitempty"` // Global templates directory
	OutputDirectory               string                      `yaml:"output_directory,omitempty"`    // Global output directory (default: ./output)
	DeviceMappings                []DeviceTemplateMapping     `yaml:"device_mappings,omitempty"`     // Global device to template mappings
	Simulators                    map[string]*SimulatorConfig `yaml:"simulators"`
	DrawIOPath                    string                      `yaml:"drawio_path,omitempty"`
	OpenKneeboardProfilesFilepath string                      `yaml:"openkneeboard_profiles_filepath,omitempty"`

	// SimpleRadio comes in exactly two installations, not one per simulator:
	// DCS-SRS and IL2-SRS are separate applications, and both IL-2 titles talk
	// to the same IL2-SRS. See SRSPathFor.
	DCSSRSPath string `yaml:"dcs_srs_path,omitempty"` // DCS-SimpleRadio-Standalone directory
	IL2SRSPath string `yaml:"il2_srs_path,omitempty"` // IL2-SimpleRadio-Standalone directory
}

// SRSPathFor returns the SimpleRadio installation a simulator's radio bindings
// come from.
//
// This is the same split srs.ParseSRSConfig already makes to find default.cfg
// (Client/default.cfg under DCS-SRS, default.cfg under IL2-SRS): the two are
// different applications, while IL-2 Great Battles and IL-2 Korea share one.
func (c *Config) SRSPathFor(simType SimulationType) string {
	if c == nil {
		return ""
	}
	if simType == DCSWorld {
		return c.DCSSRSPath
	}
	return c.IL2SRSPath
}

// GremlinsProfilePath returns the Joystick Gremlins profile configured for a
// simulator, and TargetProfilePath the Thrustmaster TARGET one.
//
// They live here beside SRSPathFor so that all the enrichers ask the
// configuration the same way. Gremlins and TARGET each carried their own copy of
// this lookup, differing only in the field read and in whether they bothered to
// check for a nil config.
//
// Neither takes a module: a Gremlins profile is set up for a whole game and a
// TARGET script for a physical HOTAS, so both apply to every module of the
// simulator they are configured on.
func (c *Config) GremlinsProfilePath(simType SimulationType) string {
	return c.simulatorProfilePath(simType, func(s *SimulatorConfig) string {
		return s.GremlinsProfileFilepath
	})
}

// TargetProfilePath returns the TARGET profile configured for a simulator.
func (c *Config) TargetProfilePath(simType SimulationType) string {
	return c.simulatorProfilePath(simType, func(s *SimulatorConfig) string {
		return s.TargetProfileFilepath
	})
}

func (c *Config) simulatorProfilePath(simType SimulationType, field func(*SimulatorConfig) string) string {
	simConfig := c.LookupSimulatorConfig(simType)
	if simConfig == nil {
		return ""
	}
	return field(simConfig)
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

// LookupSimulatorConfig returns a simulator's config without creating it, unlike
// EnsureSimulatorConfig. A nil result means the user never configured that
// simulator at all, which is different from having configured it incompletely.
func (c *Config) LookupSimulatorConfig(simType SimulationType) *SimulatorConfig {
	if c == nil || c.Simulators == nil {
		return nil
	}
	return c.Simulators[simType.GetConfigKey()]
}

// EnsureSimulatorConfig gets or creates config for a simulator
func (c *Config) EnsureSimulatorConfig(simType SimulationType) *SimulatorConfig {
	if c.Simulators == nil {
		c.Simulators = make(map[string]*SimulatorConfig)
	}

	key := simType.GetConfigKey()
	if c.Simulators[key] == nil {
		c.Simulators[key] = &SimulatorConfig{}
	}

	return c.Simulators[key]
}

// LoadConfig loads configuration from the file named by ConfigFileName, resolved
// against the current working directory.
func LoadConfig() (*Config, error) {
	return LoadConfigFrom(ConfigFileName)
}

// LoadConfigFrom loads configuration from an explicit path. A missing file yields
// an empty configuration, not an error: that is how a first run starts.
// Prefer this over LoadConfig whenever the caller is not a terminal: a windowed
// application does not control its working directory.
func LoadConfigFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
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

// SaveConfig saves configuration to the file named by ConfigFileName.
func SaveConfig(config *Config) error {
	return SaveConfigTo(config, ConfigFileName)
}

// SaveConfigTo saves configuration to an explicit path, creating the parent
// directory if needed.
func SaveConfigTo(config *Config, path string) error {
	if config == nil {
		return fmt.Errorf("no configuration to save")
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error generating YAML: %w", err)
	}

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating config directory: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
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

// UpdateDeviceTargetNumber records a controller's Thrustmaster TARGET number,
// creating the mapping if the controller has none yet.
//
// The match is exact, for the reason exactMappingIndex exists: several distinct
// controllers can share the first three GUID segments, and this used to write on
// a partial match, so two devices from the same family had their TARGET numbers
// assigned to whichever mapping came first in the slice.
// This is a helper function used when applying TARGET device mappings
func (c *Config) UpdateDeviceTargetNumber(deviceGUID, deviceName string, targetNumber int) {
	if i := c.exactMappingIndex(deviceGUID); i >= 0 {
		c.DeviceMappings[i].DeviceTargetNumber = targetNumber
		return
	}

	c.DeviceMappings = append(c.DeviceMappings, DeviceTemplateMapping{
		DeviceGUID:         deviceGUID,
		DeviceName:         deviceName,
		DeviceTargetNumber: targetNumber,
	})
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

// VerifySRSPaths checks if SRS paths are available at configured or default paths.
//
// There are two answers rather than one per simulator, because there are two
// SimpleRadio applications: see Config.SRSPathFor.
func VerifySRSPaths(config *Config) (il2Path string, dcsPath string, il2Found bool, dcsFound bool) {
	il2Path, il2Found = verifySRSPath(config.SRSPathFor(IL2Sturmovik), "C:\\Program Files\\IL2-SimpleRadio-Standalone")
	dcsPath, dcsFound = verifySRSPath(config.SRSPathFor(DCSWorld), "C:\\Program Files\\DCS-SimpleRadio-Standalone")

	return il2Path, dcsPath, il2Found, dcsFound
}

// verifySRSPath returns the configured directory when it exists, otherwise the
// stock installation directory when that one does.
func verifySRSPath(configured, defaultPath string) (string, bool) {
	if configured != "" {
		if _, err := os.Stat(configured); err == nil {
			return configured, true
		}
	}

	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, true
	}

	return "", false
}
