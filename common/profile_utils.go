package common

import (
	"cmp"
	"slices"
)

// GetAllDevicesFromProfiles extracts all unique devices from all profiles,
// virtual ones marked and sorted last.
func GetAllDevicesFromProfiles(profiles *ProfileCollection) []*Device {
	deviceMap := make(map[string]*Device)

	for _, profile := range profiles.Profiles {
		for guid, device := range profile.Devices {
			if _, exists := deviceMap[guid]; !exists {
				deviceMap[guid] = device
			}
		}
	}

	// Mark virtual devices (vJoy, Thrustmaster Combined, etc.)
	MarkVirtualDevicesInMap(deviceMap)

	var devices []*Device
	for _, device := range deviceMap {
		devices = append(devices, device)
	}

	// Sort by name for consistent display, with virtual devices at the end
	slices.SortFunc(devices, func(a, b *Device) int {
		if a.IsVirtual != b.IsVirtual {
			if a.IsVirtual {
				return 1 // Non-virtual first
			}
			return -1
		}
		return cmp.Compare(a.Name, b.Name)
	})

	return devices
}

// GroupDevicesByTemplate groups devices by their template filepath
// Returns a map: template filepath -> list of device GUIDs
func GroupDevicesByTemplate(deviceTemplates map[string]*Template) map[string][]string {
	templateGroups := make(map[string][]string)

	for deviceGUID, template := range deviceTemplates {
		if template == nil {
			continue
		}

		// Use template filepath as the key for grouping
		templateKey := template.FilePath
		templateGroups[templateKey] = append(templateGroups[templateKey], deviceGUID)
	}

	return templateGroups
}
