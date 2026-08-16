// Package app builds the parser and enricher sets every front end needs, so
// that a second entry point cannot assemble a different pipeline from the first.
//
// It used to also populate common.ExtFuncs, a set of function pointers letting
// common reach into gremlins and openkneeboard. Nothing enforced that a new
// entry point called Wire() first, and forgetting it degraded silently: devices
// known only to an external tool vanished from the export. The code that needed
// those calls moved to workflow, which can import both packages directly, and
// the indirection went with it.
package app

import (
	"simdiag/common"
	"simdiag/dcs"
	"simdiag/gremlins"
	"simdiag/il2"
	"simdiag/il2korea"
	"simdiag/openkneeboard"
	"simdiag/srs"
	"simdiag/target"
)

// Parsers returns a parser per supported simulator. IL-2 Korea ships no
// human-readable action labels, so its parser needs the config to borrow them
// from a configured Great Battles installation.
func Parsers(config *common.Config) map[common.SimulationType]common.SimulatorParser {
	return map[common.SimulationType]common.SimulatorParser{
		common.DCSWorld:     dcs.NewParser(config),
		common.IL2Sturmovik: il2.NewParser(),
		common.IL2Korea:     il2korea.NewParser(config),
	}
}

// DetectDCSModules returns the aircraft found under the configured DCS
// installation, or nil when DCS is not configured or the path does not resolve.
//
// Modules are detected rather than declared: the configuration names a DCS path
// and nothing else. Every surface that shows a module list (the CLI after the
// path is chosen, the GUI's Configuration, Devices and Generate tabs) goes
// through here so they cannot show three different answers.
//
// It reports nothing unless DCS is actually configured, even though the path
// resolution below would happily fall back to the stock Saved Games location: a
// machine with DCS installed but not configured would otherwise list modules in
// the Generate dropdown that the export then skips for want of a dcs_path.
func DetectDCSModules(config *common.Config) []string {
	if config == nil {
		return nil
	}
	if !common.SimulatorIsConfigured(common.DCSWorld, config.LookupSimulatorConfig(common.DCSWorld)) {
		return nil
	}

	modules, err := dcs.ListModules(common.ConfiguredSimulatorPath(config, common.DCSWorld))
	if err != nil {
		return nil
	}
	return modules
}

// Enrichers returns the binding enrichers in the order they must run.
func Enrichers() []common.BindingEnricher {
	return []common.BindingEnricher{
		gremlins.NewEnricher(),
		target.NewEnricher(),
		openkneeboard.NewEnricher(),
		srs.NewEnricher(),
	}
}
