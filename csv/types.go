package csv

// csvRowData holds data for a CSV row before writing
type csvRowData struct {
	simulator          string
	module             string
	action             string
	modifier           string
	modifierDevice     string // Device name for the modifier button (if different from physical device)
	modifierNum        string // Sequential number for the modifier within the module
	physicalDevice     string
	physicalInput      string
	physicalDeviceGUID string // GUID of the physical device
	virtualDevice      string
	virtualInput       string
	// templateKey is the key the binding lands on in the template, e.g.
	// "Button_6" or "AXIS_X". It is written for whoever reads the CSV; the SVG
	// importer recomputes it from Physical Input rather than trusting it, so
	// hand-editing this column changes nothing.
	templateKey  string
	templatePath string // Relative path to template file (relative to templates_directory)
}

// byColumn maps each column name to this row's value for it.
//
// The header and the values are built from this one map, so the order lives in
// AllColumns alone. They used to be two ordered lists maintained by hand in two
// files, and reordering either one would have written every value under the
// wrong header without a single test failing.
func (r *csvRowData) byColumn() map[string]string {
	return map[string]string{
		ColSimulator:          r.simulator,
		ColModule:             r.module,
		ColAction:             r.action,
		ColModifier:           r.modifier,
		ColModifierDevice:     r.modifierDevice,
		ColModifierNum:        r.modifierNum,
		ColPhysicalDevice:     r.physicalDevice,
		ColPhysicalInput:      r.physicalInput,
		ColPhysicalDeviceGUID: r.physicalDeviceGUID,
		ColVirtualDevice:      r.virtualDevice,
		ColVirtualInput:       r.virtualInput,
		ColTemplateKey:        r.templateKey,
		ColTemplate:           r.templatePath,
	}
}

// fields returns the row values in AllColumns order, ready for encoding/csv.
func (r *csvRowData) fields() []string {
	byColumn := r.byColumn()

	values := make([]string, len(AllColumns))
	for i, column := range AllColumns {
		values[i] = byColumn[column]
	}
	return values
}
