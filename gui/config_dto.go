package gui

import (
	"os"
	"path/filepath"

	"simdiag/common"
)

// The GUI speaks its own flat shape rather than marshalling common.Config
// directly: the form is per-simulator, whereas the config nests a map keyed by
// simulator, and the two simulator path fields (DCSPath, IL2InputPath) are one
// field as far as the user is concerned.

// simulatorDTO is one simulator's section of the configuration form.
type simulatorDTO struct {
	Key                     string `json:"key"`   // dcs_world, il2_sturmovik, il2_korea
	Label                   string `json:"label"` // human-readable simulator name
	Path                    string `json:"path"`  // DCSPath or IL2InputPath
	GremlinsProfileFilepath string `json:"gremlinsProfileFilepath"`
	TargetProfileFilepath   string `json:"targetProfileFilepath"`
}

// configDTO is the whole configuration form.
//
// The two SRS paths are global rather than per-simulator, like the config they
// come from: there are two SimpleRadio installations, and both IL-2 titles share
// one. The form renders the IL-2 one once, under a section covering both.
type configDTO struct {
	TemplatesDirectory            string         `json:"templatesDirectory"`
	OutputDirectory               string         `json:"outputDirectory"`
	DrawIOPath                    string         `json:"drawioPath"`
	OpenKneeboardProfilesFilepath string         `json:"openkneeboardProfilesFilepath"`
	DCSSRSPath                    string         `json:"dcsSrsPath"`
	IL2SRSPath                    string         `json:"il2SrsPath"`
	Simulators                    []simulatorDTO `json:"simulators"`
}

// pathStatus reports whether a configured path actually resolves on disk.
type pathStatus struct {
	Path     string  `json:"path"`
	Exists   bool    `json:"exists"`
	Detail   message `json:"detail,omitzero"`
	Severity string  `json:"severity,omitempty"` // "", "warn" or "error"
}

// simulatorStatus is the per-simulator half of configStatus.
type simulatorStatus struct {
	Key      string     `json:"key"`
	Path     pathStatus `json:"path"`
	Gremlins pathStatus `json:"gremlins"`
	Target   pathStatus `json:"target"`
	// Modules lists the DCS aircraft found under the configured path. It is
	// detection, not configuration: nothing in the YAML names a module.
	Modules []string `json:"modules"`
}

// baseTemplatesStatus reports what the binary could add to the templates
// directory: SimDiag ships the diagrams of the common controllers, and a fresh
// install has nowhere to draw on.
type baseTemplatesStatus struct {
	// Missing is always a list, never null: the frontend reads its length first.
	Missing []string `json:"missing"`
	Total   int      `json:"total"`
	Target  string   `json:"target"` // where they would be written
}

// configStatus tells the form which paths resolve, so the user finds out before
// running an export rather than during it.
type configStatus struct {
	Templates     pathStatus                 `json:"templates"`
	BaseTemplates baseTemplatesStatus        `json:"baseTemplates"`
	Output        pathStatus                 `json:"output"`
	DrawIO        pathStatus                 `json:"drawio"`
	OpenKneeboard pathStatus                 `json:"openkneeboard"`
	DCSSRS        pathStatus                 `json:"dcsSrs"`
	IL2SRS        pathStatus                 `json:"il2Srs"`
	Simulators    map[string]simulatorStatus `json:"simulators"`
}

// simulatorOrder fixes the order simulators appear in the form.
var simulatorOrder = []common.SimulationType{
	common.DCSWorld,
	common.IL2Sturmovik,
	common.IL2Korea,
}

// toDTO projects the configuration onto the form shape.
func toDTO(config *common.Config) configDTO {
	dto := configDTO{
		TemplatesDirectory:            config.TemplatesDirectory,
		OutputDirectory:               config.OutputDirectory,
		DrawIOPath:                    config.DrawIOPath,
		OpenKneeboardProfilesFilepath: config.OpenKneeboardProfilesFilepath,
		DCSSRSPath:                    config.DCSSRSPath,
		IL2SRSPath:                    config.IL2SRSPath,
	}

	for _, simType := range simulatorOrder {
		sim := simulatorDTO{Key: simType.GetConfigKey(), Label: string(simType)}

		if section := config.LookupSimulatorConfig(simType); section != nil {
			if simType == common.DCSWorld {
				sim.Path = section.DCSPath
			} else {
				sim.Path = section.IL2InputPath
			}
			sim.GremlinsProfileFilepath = section.GremlinsProfileFilepath
			sim.TargetProfileFilepath = section.TargetProfileFilepath
		}

		dto.Simulators = append(dto.Simulators, sim)
	}

	return dto
}

// applyDTO writes the form back onto the configuration, leaving everything the
// form does not cover (device mappings, DCS modules) untouched.
func applyDTO(config *common.Config, dto configDTO) {
	config.TemplatesDirectory = dto.TemplatesDirectory
	config.OutputDirectory = dto.OutputDirectory
	config.DrawIOPath = dto.DrawIOPath
	config.OpenKneeboardProfilesFilepath = dto.OpenKneeboardProfilesFilepath
	config.DCSSRSPath = dto.DCSSRSPath
	config.IL2SRSPath = dto.IL2SRSPath

	for _, sim := range dto.Simulators {
		simType, ok := simulatorTypeForKey(sim.Key)
		if !ok {
			continue
		}

		// Nothing configured for this simulator and no section yet: do not
		// create one, so the YAML keeps only what the user actually set.
		if config.LookupSimulatorConfig(simType) == nil && sim.isEmpty() {
			continue
		}

		section := config.EnsureSimulatorConfig(simType)
		if simType == common.DCSWorld {
			section.DCSPath = sim.Path
		} else {
			section.IL2InputPath = sim.Path
		}
		section.GremlinsProfileFilepath = sim.GremlinsProfileFilepath
		section.TargetProfileFilepath = sim.TargetProfileFilepath
	}
}

func (s simulatorDTO) isEmpty() bool {
	return s.Path == "" && s.GremlinsProfileFilepath == "" && s.TargetProfileFilepath == ""
}

func simulatorTypeForKey(key string) (common.SimulationType, bool) {
	for _, simType := range simulatorOrder {
		if simType.GetConfigKey() == key {
			return simType, true
		}
	}
	return "", false
}

// defaultsDTO suggests stock locations for everything the user has not set yet,
// so a first run is a matter of confirming rather than typing.
func defaultsDTO(config *common.Config) configDTO {
	dto := configDTO{
		TemplatesDirectory: "templates",
		OutputDirectory:    "output",
	}

	if path, found := common.VerifyDrawIOPath(config); found {
		dto.DrawIOPath = path
	}
	dto.OpenKneeboardProfilesFilepath = filepath.Join(os.Getenv("LOCALAPPDATA"), "OpenKneeboard", "Settings", "Profiles.json")

	dto.IL2SRSPath, dto.DCSSRSPath, _, _ = common.VerifySRSPaths(config)

	for _, simType := range simulatorOrder {
		dto.Simulators = append(dto.Simulators, simulatorDTO{
			Key:   simType.GetConfigKey(),
			Label: string(simType),
			Path:  common.DefaultSimulatorPath(simType),
		})
	}

	return dto
}
