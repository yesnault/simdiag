package gui

// message is prose the frontend renders.
//
// The interface is bilingual, and the wording lives in one place: the catalogue
// in gui/frontend/i18n.js. So Go names a message and supplies its values instead
// of writing the sentence: French does not compose in the same order as English,
// and a language switch has to re-render what is already on screen without
// asking the server again.
//
// Real Go errors travel as an argument, not as a code: they are the system
// speaking, not the interface.
type message struct {
	Code string            `json:"code"`
	Args map[string]string `json:"args,omitempty"`
}

// msg builds a message with no arguments.
func msg(code string) message {
	return message{Code: code}
}

// msgArgs builds a message with values to substitute into the translation.
func msgArgs(code string, args map[string]string) message {
	return message{Code: code, Args: args}
}

// errorArg is the conventional argument name for a wrapped Go error.
func errorArg(err error) map[string]string {
	return map[string]string{"error": err.Error()}
}

// Message codes. Every one of these must have an entry in the frontend
// catalogue. gui/i18n_test.go fails the build if one is missing.
const (
	// Configuration tab status details.
	msgNotConfigured          = "status.notConfigured"
	msgTemplatesRequired      = "status.templates.required"
	msgTemplatesDirNotFound   = "status.templates.dirNotFound"
	msgTemplatesUnreadable    = "status.templates.unreadable"
	msgTemplatesNone          = "status.templates.none"
	msgTemplatesFound         = "status.templates.found"
	msgOutputRequired         = "status.output.required"
	msgOutputWillBeCreated    = "status.output.willBeCreated"
	msgDrawIONotFound         = "status.drawio.notFound"
	msgOpenKneeboardNotFound  = "status.openkneeboard.notFound"
	msgSRSNotFound            = "status.srs.notFound"
	msgGremlinsNotFound       = "status.gremlins.notFound"
	msgTargetNotFound         = "status.target.notFound"
	msgSimulatorNotConfigured = "status.simulator.notConfigured"
	msgSimulatorDirNotFound   = "status.simulator.dirNotFound"

	// Devices tab.
	msgParserFailed          = "devices.parserFailed"
	msgTemplatesUnconfigured = "devices.templatesUnconfigured"
	msgTemplatesReadFailed   = "devices.templatesReadFailed"
	msgTargetDetectFailed    = "error.targetDetectFailed"

	// Diagrams tab.
	msgOutputUnconfigured = "diagrams.outputUnconfigured"
	msgOutputReadFailed   = "diagrams.outputReadFailed"
	msgNoDiagramYet       = "diagrams.noneYet"
	msgOutputGroup        = "diagrams.outputGroup"

	// Generate tab.
	msgExportTargetEverything = "generate.targetEverything"
	msgExportNotConfigured    = "generate.notConfigured"

	// Refusals the user is meant to act on.
	msgExportRunning = "error.exportRunning"
	msgNoConfigFile  = "error.noConfigFile"
	msgNoCSVToRegen  = "error.noCSVToRegenerate"
	msgFolderMissing = "error.folderMissing"

	// Tips tab.
	msgBatchScriptFailed = "error.batchScriptFailed"

	// About tab.
	msgUpdateCheckFailed = "error.updateCheckFailed"
	msgUpdateRunning     = "error.updateRunning"
	msgAlreadyCurrent    = "error.alreadyCurrent"
	msgNothingInstalled  = "error.nothingInstalled"
	msgRestartFailed     = "error.restartFailed"
)
