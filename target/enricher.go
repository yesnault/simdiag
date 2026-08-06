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
