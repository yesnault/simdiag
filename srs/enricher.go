package srs

import "simdiag/common"

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
	if simConfig == nil || simConfig.SRSPath == "" {
		return // SRS not configured for this simulator
	}
	srsPath := simConfig.SRSPath

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
