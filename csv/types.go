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
	templateKey        string // Template key that will be replaced (e.g., "Button_6", "AXIS_X")
	templatePath       string // Relative path to template file (relative to templates_directory)
}

// fields returns the row values in AllColumns order, ready for encoding/csv.
func (r *csvRowData) fields() []string {
	return []string{
		r.simulator,
		r.module,
		r.action,
		r.modifier,
		r.modifierDevice,
		r.modifierNum,
		r.physicalDevice,
		r.physicalInput,
		r.physicalDeviceGUID,
		r.virtualDevice,
		r.virtualInput,
		r.templateKey,
		r.templatePath,
	}
}
