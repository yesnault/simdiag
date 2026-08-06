package common

import (
	"sort"
)

// AssignModifierNumbers assigns sequential modifier numbers to all bindings based on their modifiers.
// This should be called after all enrichments are done, before exporting to CSV/SVG.
func AssignModifierNumbers(exportDevices []*ExportDevice) {
	modifiersByModule := collectModifiersByModule(exportDevices)
	modifierNumbers := createModifierNumberMappings(modifiersByModule)
	applyModifierNumbers(exportDevices, modifierNumbers)
}

// collectModifiersByModule collects all unique modifier keys grouped by module
func collectModifiersByModule(exportDevices []*ExportDevice) map[string]map[string]bool {
	modifiersByModule := make(map[string]map[string]bool)

	for _, exportDevice := range exportDevices {
		module := getModuleName(exportDevice)
		if module == "" {
			continue
		}

		if modifiersByModule[module] == nil {
			modifiersByModule[module] = make(map[string]bool)
		}

		collectModifierKeysFromBindings(exportDevice.Profile.Bindings, modifiersByModule[module])
	}

	return modifiersByModule
}

// getModuleName extracts the module name from an export device
func getModuleName(exportDevice *ExportDevice) string {
	if exportDevice.Profile.SimType == DCSWorld && exportDevice.Profile.Module != "" {
		return exportDevice.Profile.Module
	}
	if exportDevice.Profile.SimType == IL2Sturmovik {
		return "il2"
	}
	if exportDevice.Profile.SimType == IL2Korea {
		return "il2-korea"
	}
	return ""
}

// collectModifierKeysFromBindings collects all modifier keys from bindings
func collectModifierKeysFromBindings(bindings []Binding, modifierSet map[string]bool) {
	for _, binding := range bindings {
		// Collect keys from bindings that USE modifiers
		if len(binding.Modifiers) > 0 {
			for _, mod := range binding.Modifiers {
				for _, key := range mod.Keys {
					modifierSet[key] = true
				}
			}
		}

		// Collect keys from bindings that ARE modifier definitions
		if binding.ModifierKey != "" {
			modifierSet[binding.ModifierKey] = true
		}
	}
}

// createModifierNumberMappings creates sorted number mappings for each module
func createModifierNumberMappings(modifiersByModule map[string]map[string]bool) map[string]map[string]int {
	modifierNumbers := make(map[string]map[string]int)

	for module, modifierSet := range modifiersByModule {
		modifierNumbers[module] = assignNumbersToModifiers(modifierSet)
	}

	return modifierNumbers
}

// assignNumbersToModifiers converts a set of modifier keys to a sorted number mapping
func assignNumbersToModifiers(modifierSet map[string]bool) map[string]int {
	modifiers := make([]string, 0, len(modifierSet))
	for key := range modifierSet {
		modifiers = append(modifiers, key)
	}
	sort.Strings(modifiers)

	numbering := make(map[string]int)
	for i, key := range modifiers {
		numbering[key] = i + 1
	}

	return numbering
}

// applyModifierNumbers assigns modifier numbers to all bindings
func applyModifierNumbers(exportDevices []*ExportDevice, modifierNumbers map[string]map[string]int) {
	for _, exportDevice := range exportDevices {
		module := getModuleName(exportDevice)
		if module == "" {
			continue
		}

		modMap := modifierNumbers[module]
		if modMap == nil {
			continue
		}

		assignNumbersToProfileBindings(exportDevice.Profile.Bindings, modMap)
	}
}

// assignNumbersToProfileBindings assigns numbers to bindings in a profile
func assignNumbersToProfileBindings(bindings []Binding, modMap map[string]int) {
	for i := range bindings {
		binding := &bindings[i]

		// Assign to bindings that USE modifiers
		if len(binding.Modifiers) > 0 && len(binding.Modifiers[0].Keys) > 0 {
			firstModifierKey := binding.Modifiers[0].Keys[0]
			if num, exists := modMap[firstModifierKey]; exists {
				binding.ModifierNum = num
			}
		}

		// Assign to bindings that ARE modifier definitions
		if binding.ModifierKey != "" {
			if num, exists := modMap[binding.ModifierKey]; exists {
				binding.ModifierNum = num
			}
		}
	}
}
