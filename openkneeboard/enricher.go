package openkneeboard

import (
	"os"
	"simdiag/common"
)

// Enricher implements the BindingEnricher interface for OpenKneeboard
type Enricher struct{}

// NewEnricher creates a new OpenKneeboard enricher instance
func NewEnricher() *Enricher {
	return &Enricher{}
}

// Enrich adds OpenKneeboard bindings to an ExportDevice
func (e *Enricher) Enrich(exportDevice *common.ExportDevice, _ *common.Profile, config *common.Config) {
	addBindings(exportDevice, config)
}

// GetName returns the human-readable name of the enricher
func (e *Enricher) GetName() string {
	return "OpenKneeboard"
}

// IsAvailable checks if OpenKneeboard profile is configured and available
func (e *Enricher) IsAvailable(config *common.Config) bool {
	if config == nil {
		return false
	}

	// Check if OpenKneeboard profiles path is configured
	if config.OpenKneeboardProfilesFilepath == "" {
		return false
	}

	// Check if file exists
	if _, err := os.Stat(config.OpenKneeboardProfilesFilepath); os.IsNotExist(err) {
		return false
	}

	return true
}
