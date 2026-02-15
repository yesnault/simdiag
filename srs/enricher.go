package srs

import (
	"os"
	"path/filepath"
	"simdiag/common"
)

// Enricher implements the BindingEnricher interface for SRS (SimpleRadio Standalone)
type Enricher struct{}

// NewEnricher creates a new SRS enricher instance
func NewEnricher() *Enricher {
	return &Enricher{}
}

// Enrich adds SRS bindings to an ExportDevice
func (e *Enricher) Enrich(exportDevice *common.ExportDevice, fullProfile *common.Profile, config *common.Config) {
	// Get SRS path from simulator config
	simConfig := config.GetSimulatorConfig(fullProfile.SimType)
	srsPath := simConfig.SRSPath

	if srsPath == "" {
		return // SRS not configured for this simulator
	}

	// Parse SRS config
	bindingsByDevice, err := ParseSRSConfig(srsPath, fullProfile.SimType)
	if err != nil {
		return // Silently fail - SRS is optional
	}

	// Add SRS bindings for all devices in the profile (handles merged devices)
	for deviceGUID := range exportDevice.Profile.Devices {
		// Normalize the device GUID to match SRS parser format (lowercase)
		normalizedGUID := common.NormalizeGUID(deviceGUID)

		if deviceBindings, exists := bindingsByDevice[normalizedGUID]; exists {
			exportDevice.Profile.Bindings = append(exportDevice.Profile.Bindings, deviceBindings...)
		}
	}
}

// GetName returns the human-readable name of the enricher
func (e *Enricher) GetName() string {
	return "SRS (SimpleRadio Standalone)"
}

// IsAvailable checks if SRS is configured and available
func (e *Enricher) IsAvailable(config *common.Config) bool {
	if config == nil {
		return false
	}

	// Check if at least one simulator has SRS configured
	if dcsConfig := config.GetSimulatorConfig(common.DCSWorld); dcsConfig != nil && dcsConfig.SRSPath != "" {
		configPath := filepath.Join(dcsConfig.SRSPath, "default.cfg")
		if _, err := os.Stat(configPath); err == nil {
			return true
		}
	}

	if il2Config := config.GetSimulatorConfig(common.IL2Sturmovik); il2Config != nil && il2Config.SRSPath != "" {
		configPath := filepath.Join(il2Config.SRSPath, "default.cfg")
		if _, err := os.Stat(configPath); err == nil {
			return true
		}
	}

	return false
}
