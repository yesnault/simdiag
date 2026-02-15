package gremlins

import "simdiag/common"

// Enricher implements the BindingEnricher interface for Gremlins
type Enricher struct{}

// NewEnricher creates a new Gremlins enricher instance
func NewEnricher() *Enricher {
	return &Enricher{}
}

// Enrich adds Gremlins bindings to an ExportDevice
func (e *Enricher) Enrich(exportDevice *common.ExportDevice, fullProfile *common.Profile, config *common.Config) {
	addBindings(exportDevice, fullProfile, config)
}

// GetName returns the human-readable name of the enricher
func (e *Enricher) GetName() string {
	return "Gremlins"
}

// IsAvailable checks if Gremlins profile is configured and available
func (e *Enricher) IsAvailable(config *common.Config) bool {
	if config == nil {
		return false
	}

	// Gremlins availability is checked per-device during enrichment
	// Just return true here - actual availability is determined in AddGremlinsBindings
	return true
}
