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
	srsPath := config.SRSPathFor(fullProfile.SimType)
	if srsPath == "" {
		return // SRS not configured
	}

	// Parse SRS config
	bindingsByDevice, err := ParseSRSConfig(srsPath, fullProfile.SimType)
	if err != nil {
		return // Silently fail - SRS is optional
	}

	// Add SRS bindings for all devices in the profile (handles merged devices).
	//
	// The match has to be partial: SRS records 5-segment GUIDs while IL-2 uses
	// 4-segment ones, so exact equality silently dropped every IL-2 radio
	// binding. And the binding takes the simulator's own device identity rather
	// than the one SRS saw, for the same reason OpenKneeboard does: the CSV
	// writes DeviceGUID as the physical device, and it has to be the GUID the
	// rest of that simulator's export uses.
	for deviceGUID := range exportDevice.Profile.Devices {
		for srsGUID, srsBindings := range bindingsByDevice {
			if !common.MatchGUIDPartial(deviceGUID, srsGUID) {
				continue
			}

			for _, binding := range srsBindings {
				binding.DeviceGUID = exportDevice.Device.GUID
				binding.DeviceName = exportDevice.Device.Name
				exportDevice.Profile.Bindings = append(exportDevice.Profile.Bindings, binding)
			}
		}
	}
}
