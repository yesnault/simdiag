package gui

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"simdiag/app"
	"simdiag/common"
	"simdiag/templates"
)

// computeStatus checks every configured path against the filesystem. The point
// is to surface a wrong path while the user is looking at the form, instead of
// halfway through an export.
//
// Every detail is a message code rather than a sentence: the wording, in both
// languages, belongs to the frontend catalogue. Note that each optional path
// gets its own code instead of a shared "not found" glued to a consequence,
// French does not reassemble in the same order as English.
func computeStatus(config *common.Config) configStatus {
	status := configStatus{
		Templates:     templatesStatus(config.TemplatesDirectory),
		BaseTemplates: baseTemplatesStatusFor(config),
		Output:        outputStatus(config.OutputDirectory),
		DrawIO:        drawIOStatus(config),
		OpenKneeboard: optionalFileStatus(config.OpenKneeboardProfilesFilepath, msgOpenKneeboardNotFound),
		DCSSRS:        optionalDirectoryStatus(config.DCSSRSPath, msgSRSNotFound),
		IL2SRS:        optionalDirectoryStatus(config.IL2SRSPath, msgSRSNotFound),
		Simulators:    make(map[string]simulatorStatus, len(simulatorOrder)),
	}

	for _, simType := range simulatorOrder {
		status.Simulators[simType.GetConfigKey()] = simulatorStatusFor(config, simType)
	}

	return status
}

// defaultTemplatesDirectory is where the base templates go when nothing is
// configured yet: beside the configuration file, which the process has already
// moved to (enterConfigDirectory). It is also what defaultsDTO suggests, so the
// banner and the field agree.
const defaultTemplatesDirectory = "templates"

// baseTemplatesStatusFor reports which of the templates shipped in the binary
// are not on disk yet, and where they would be written.
func baseTemplatesStatusFor(config *common.Config) baseTemplatesStatus {
	target := config.TemplatesDirectory
	if target == "" {
		target = defaultTemplatesDirectory
	}

	missing := templates.Missing(target)
	if missing == nil {
		missing = []string{}
	}

	return baseTemplatesStatus{Missing: missing, Total: len(templates.Names()), Target: target}
}

func templatesStatus(dir string) pathStatus {
	if dir == "" {
		return pathStatus{Detail: msg(msgTemplatesRequired), Severity: "error"}
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return pathStatus{Path: dir, Detail: msg(msgTemplatesDirNotFound), Severity: "error"}
	}

	count, err := countSVGFiles(dir)
	if err != nil {
		return pathStatus{Path: dir, Exists: true, Detail: msgArgs(msgTemplatesUnreadable, errorArg(err)), Severity: "error"}
	}
	if count == 0 {
		return pathStatus{Path: dir, Exists: true, Detail: msg(msgTemplatesNone), Severity: "error"}
	}

	return pathStatus{
		Path:   dir,
		Exists: true,
		Detail: msgArgs(msgTemplatesFound, map[string]string{"count": strconv.Itoa(count)}),
	}
}

func outputStatus(dir string) pathStatus {
	if dir == "" {
		return pathStatus{Detail: msg(msgOutputRequired), Severity: "error"}
	}

	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return pathStatus{Path: dir, Exists: true}
	}

	// Not an error: the export creates the directory.
	return pathStatus{Path: dir, Detail: msg(msgOutputWillBeCreated)}
}

func drawIOStatus(config *common.Config) pathStatus {
	if path, found := common.VerifyDrawIOPath(config); found {
		return pathStatus{Path: path, Exists: true}
	}

	return pathStatus{
		Path:     config.DrawIOPath,
		Detail:   msg(msgDrawIONotFound),
		Severity: "warn",
	}
}

// optionalFileStatus reports on a file whose absence only disables a feature.
// missingCode names the message describing what will be missing.
func optionalFileStatus(path, missingCode string) pathStatus {
	if path == "" {
		return pathStatus{Detail: msg(msgNotConfigured), Severity: "warn"}
	}
	if _, err := os.Stat(path); err != nil {
		return pathStatus{Path: path, Detail: msg(missingCode), Severity: "warn"}
	}
	return pathStatus{Path: path, Exists: true}
}

// optionalDirectoryStatus is optionalFileStatus for a setting that names a
// directory (an SRS installation, not a file inside it) so that a wrong path
// is not reported as a missing file.
func optionalDirectoryStatus(path, missingCode string) pathStatus {
	if path == "" {
		return pathStatus{Detail: msg(msgNotConfigured), Severity: "warn"}
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return pathStatus{Path: path, Detail: msg(missingCode), Severity: "warn"}
	}
	return pathStatus{Path: path, Exists: true}
}

func simulatorStatusFor(config *common.Config, simType common.SimulationType) simulatorStatus {
	status := simulatorStatus{Key: simType.GetConfigKey()}
	section := config.LookupSimulatorConfig(simType)

	if section == nil {
		status.Path = pathStatus{Detail: msg(msgNotConfigured), Severity: "warn"}
		return status
	}

	configured := section.DCSPath
	if simType != common.DCSWorld {
		configured = section.IL2InputPath
	}

	switch {
	case configured == "":
		status.Path = pathStatus{Detail: msg(msgSimulatorNotConfigured), Severity: "warn"}
	default:
		if info, err := os.Stat(configured); err == nil && info.IsDir() {
			status.Path = pathStatus{Path: configured, Exists: true}
		} else {
			status.Path = pathStatus{Path: configured, Detail: msg(msgSimulatorDirNotFound), Severity: "error"}
		}
	}

	status.Gremlins = optionalFileStatus(section.GremlinsProfileFilepath, msgGremlinsNotFound)
	status.Target = optionalFileStatus(section.TargetProfileFilepath, msgTargetNotFound)

	// Detected, not configured: the user picks a DCS path and the aircraft found
	// under it are what will be exported.
	if simType == common.DCSWorld {
		status.Modules = app.DetectDCSModules(config)
	}

	return status
}

// countSVGFiles counts templates without reading them. common.FindTemplates
// parses every file, which means loading ~11 MB just to answer "how many?".
func countSVGFiles(dir string) (int, error) {
	count := 0
	err := filepath.WalkDir(dir, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".svg") {
			count++
		}
		return nil
	})
	return count, err
}
