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
