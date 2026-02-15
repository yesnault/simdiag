package common

import (
	"strings"
)

// VirtualDevicePatterns contains patterns to detect virtual devices by name
// These devices don't need templates as they are software-created virtual devices
var VirtualDevicePatterns = []string{
	"vjoy",                  // vJoy virtual devices
	"virtual",               // Generic virtual devices
	"combined",              // Thrustmaster TARGET combined device
	"thrustmaster combined", // Explicit TARGET combined device
	"vjoy device",           // vJoy device with space
}

// IsVirtualDevice checks if a device is virtual based on its name
// Virtual devices are created by software like Gremlins (vJoy) or TARGET (Combined)
// and don't need diagram templates
func IsVirtualDevice(deviceName string) bool {
	nameLower := strings.ToLower(deviceName)

	for _, pattern := range VirtualDevicePatterns {
		if strings.Contains(nameLower, pattern) {
			return true
		}
	}

	return false
}

// IsVirtualDeviceGUID checks if a device GUID matches known virtual device patterns
// Some virtual devices have recognizable GUID patterns
func IsVirtualDeviceGUID(deviceGUID string) bool {
	guidLower := strings.ToLower(deviceGUID)

	// vJoy devices often have specific GUID patterns
	// Example: {a768ef40-b302-11ea-...} for vJoy Device 1
	// The GUID contains "b302-11ea" or similar vJoy patterns
	if strings.Contains(guidLower, "b302-11ea") || strings.Contains(guidLower, "b310-11ea") {
		return true
	}

	return false
}

// MarkVirtualDevicesInMap marks devices as virtual in a map structure
func MarkVirtualDevicesInMap(devices map[string]*Device) {
	for _, device := range devices {
		if device == nil {
			continue
		}
		device.IsVirtual = IsVirtualDevice(device.Name) || IsVirtualDeviceGUID(device.GUID)
	}
}

// FilterPhysicalDevices returns only physical (non-virtual) devices
func FilterPhysicalDevices(devices []*Device) []*Device {
	physical := make([]*Device, 0)
	for _, device := range devices {
		if device != nil && !device.IsVirtual {
			physical = append(physical, device)
		}
	}
	return physical
}
