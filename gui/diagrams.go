package gui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"simdiag/common"
)

// diagramEntry is one generated diagram, with its PNG copy when draw.io produced one.
type diagramEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // relative to the output directory
	PNGPath string `json:"pngPath,omitempty"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
}

// diagramGroup is one output subdirectory: a DCS module, or a simulator.
//
// Label is a proper noun and is shown as is; LabelCode names a translated
// heading and wins when set. Only the output directory itself needs one.
type diagramGroup struct {
	Label     string         `json:"label"`
	LabelCode string         `json:"labelCode,omitempty"`
	Directory string         `json:"directory"` // relative to the output directory
	Diagrams  []diagramEntry `json:"diagrams"`
}

// diagramsPayload is what the Diagrams tab renders.
type diagramsPayload struct {
	Groups     []diagramGroup `json:"groups"`
	OutputPath string         `json:"outputPath"`
	CSVPath    string         `json:"csvPath"`
	HasCSV     bool           `json:"hasCsv"`
	Warnings   []message      `json:"warnings"`
}

// scanDiagrams lists what the exports actually left on disk, following the
// layout common.OutputSubdir produces: dcs-<module>/, il2/, il2-korea/, each
// holding the SVG files and a png/ subdirectory.
func scanDiagrams(config *common.Config) diagramsPayload {
	payload := diagramsPayload{Groups: []diagramGroup{}}

	if config.OutputDirectory == "" {
		payload.Warnings = append(payload.Warnings, msg(msgOutputUnconfigured))
		return payload
	}

	absOutput, err := filepath.Abs(config.OutputDirectory)
	if err == nil {
		payload.OutputPath = absOutput
	} else {
		payload.OutputPath = config.OutputDirectory
	}

	csvPath := filepath.Join(config.OutputDirectory, "export.csv")
	payload.CSVPath = csvPath
	payload.HasCSV = fileExists(csvPath)

	entries, err := os.ReadDir(config.OutputDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			payload.Warnings = append(payload.Warnings, msg(msgNoDiagramYet))
			return payload
		}
		payload.Warnings = append(payload.Warnings, msgArgs(msgOutputReadFailed, errorArg(err)))
		return payload
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if group, ok := readDiagramGroup(config.OutputDirectory, entry.Name()); ok {
			payload.Groups = append(payload.Groups, group)
		}
	}

	// Diagrams may also sit directly in the output directory when a simulator
	// has no subdirectory of its own.
	if root, ok := readDiagramGroup(config.OutputDirectory, ""); ok {
		payload.Groups = append(payload.Groups, root)
	}

	slices.SortFunc(payload.Groups, func(a, b diagramGroup) int {
		return strings.Compare(a.Label, b.Label)
	})

	if len(payload.Groups) == 0 {
		payload.Warnings = append(payload.Warnings, msg(msgNoDiagramYet))
	}

	return payload
}

// readDiagramGroup collects the SVG files of one output subdirectory.
func readDiagramGroup(outputDir, subdir string) (diagramGroup, bool) {
	dir := filepath.Join(outputDir, subdir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return diagramGroup{}, false
	}

	group := diagramGroup{
		Label:     diagramGroupLabel(subdir),
		LabelCode: diagramGroupLabelCode(subdir),
		Directory: subdir,
		Diagrams:  []diagramEntry{},
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".svg") {
			continue
		}

		diagram := diagramEntry{
			Name: entry.Name(),
			Path: filepath.ToSlash(filepath.Join(subdir, entry.Name())),
		}
		if info, err := entry.Info(); err == nil {
			diagram.Size = info.Size()
			diagram.ModTime = info.ModTime().Format("2006-01-02 15:04")
		}

		// The PNG copy lives in a png/ subdirectory, same basename.
		png := filepath.Join(dir, "png", strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))+".png")
		if fileExists(png) {
			diagram.PNGPath = filepath.ToSlash(filepath.Join(subdir, "png",
				strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))+".png"))
		}

		group.Diagrams = append(group.Diagrams, diagram)
	}

	if len(group.Diagrams) == 0 {
		return diagramGroup{}, false
	}

	slices.SortFunc(group.Diagrams, func(a, b diagramEntry) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return group, true
}

// diagramGroupLabel turns an output subdirectory name back into something
// readable, mirroring common.OutputSubdir.
func diagramGroupLabel(subdir string) string {
	switch {
	case subdir == "":
		return "" // translated, see diagramGroupLabelCode
	case subdir == "il2":
		return string(common.IL2Sturmovik)
	case subdir == "il2-korea":
		return string(common.IL2Korea)
	case strings.HasPrefix(subdir, "dcs-"):
		return string(common.DCSWorld) + " · " + strings.TrimPrefix(subdir, "dcs-")
	}
	return subdir
}

// diagramGroupLabelCode names the translated heading of a group, for the one
// group that is not named after a simulator: the output directory itself.
func diagramGroupLabelCode(subdir string) string {
	if subdir == "" {
		return msgOutputGroup
	}
	return ""
}
