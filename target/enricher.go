package target

import "simdiag/common"

// Enricher implements the BindingEnricher interface for TARGET
type Enricher struct{}

// NewEnricher creates a new TARGET enricher instance
func NewEnricher() *Enricher {
	return &Enricher{}
}

// Enrich adds TARGET bindings to an ExportDevice
func (e *Enricher) Enrich(exportDevice *common.ExportDevice, fullProfile *common.Profile, config *common.Config) {
	addBindings(exportDevice, fullProfile, config)
}

// GetName returns the human-readable name of the enricher
func (e *Enricher) GetName() string {
	return "TARGET"
}

// IsAvailable checks if TARGET profile is configured and available
func (e *Enricher) IsAvailable(config *common.Config) bool {
	if config == nil {
		return false
	}

	// TARGET availability is checked per-device during enrichment
	// Just return true here - actual availability is determined in AddTargetBindings
	return true
}
