package openkneeboard

import "simdiag/common"

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
