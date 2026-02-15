package openkneeboard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"simdiag/common"
	"strings"
)

// Profiles represents the Profiles.json structure
type Profiles struct {
	ActiveProfile  string        `json:"ActiveProfile"`
	DefaultProfile string        `json:"DefaultProfile"`
	Enabled        bool          `json:"Enabled"`
	LoopProfiles   bool          `json:"LoopProfiles"`
	Profiles       []ProfileInfo `json:"Profiles"`
}

// ProfileInfo represents a profile entry
type ProfileInfo struct {
	GUID string `json:"Guid"`
	Name string `json:"Name"`
}

// DirectInput represents the DirectInput.json structure
type DirectInput struct {
	Devices map[string]Device `json:"Devices"`
}

// Device represents a device configuration
type Device struct {
	ID             string          `json:"ID"`
	Kind           string          `json:"Kind"`
	Name           string          `json:"Name"`
	ButtonBindings []ButtonBinding `json:"ButtonBindings,omitempty"`
}

// ButtonBinding represents a button binding
type ButtonBinding struct {
	Action  string `json:"Action"`
	Buttons []int  `json:"Buttons"`
}

// Binding represents a processed binding for export
type Binding struct {
	DeviceGUID string
	DeviceName string
	InputType  common.InputType // Button
	InputID    string           // "button_0", "button_1", etc.
	Action     string           // "PREVIOUS_TAB", "NEXT_TAB", etc.
}

// ParseProfiles reads and parses the Profiles.json file
func ParseProfiles(profilesPath string) (*Profiles, error) {
	data, err := os.ReadFile(profilesPath)
	if err != nil {
		return nil, fmt.Errorf("error reading Profiles.json: %w", err)
	}

	var profiles Profiles
	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, fmt.Errorf("error parsing Profiles.json: %w", err)
	}

	return &profiles, nil
}

// ParseDirectInput reads and parses the DirectInput.json file
func ParseDirectInput(directInputPath string) (*DirectInput, error) {
	data, err := os.ReadFile(directInputPath)
	if err != nil {
		return nil, fmt.Errorf("error reading DirectInput.json: %w", err)
	}

	var directInput DirectInput
	if err := json.Unmarshal(data, &directInput); err != nil {
		return nil, fmt.Errorf("error parsing DirectInput.json: %w", err)
	}

	return &directInput, nil
}

// LoadBindings loads OpenKneeboard bindings for a specific device
func LoadBindings(profilesPath, deviceGUID string) []*Binding {
	if profilesPath == "" {
		return nil
	}

	// Parse Profiles.json to get the default profile GUID
	profiles, err := ParseProfiles(profilesPath)
	if err != nil {
		fmt.Printf("⚠ Error parsing OpenKneeboard Profiles.json: %v\n", err)
		return nil
	}

	// Use DefaultProfile to find the DirectInput.json file
	profileGUID := profiles.DefaultProfile
	if profileGUID == "" {
		return nil
	}

	// Build path to DirectInput.json: {Profiles.json directory}/Profiles/{GUID}/DirectInput.json
	profilesDir := filepath.Dir(profilesPath)
	directInputPath := filepath.Join(profilesDir, "Profiles", strings.Trim(profileGUID, "{}"), "DirectInput.json")

	// Parse DirectInput.json
	directInput, err := ParseDirectInput(directInputPath)
	if err != nil {
		fmt.Printf("⚠ Error parsing OpenKneeboard DirectInput.json: %v\n", err)
		return nil
	}

	// Normalize device GUID for comparison
	normalizedDeviceGUID := common.NormalizeGUIDUpper(deviceGUID)

	// Find bindings for this device
	deviceBindings := make([]*Binding, 0)
	for guid, device := range directInput.Devices {
		normalizedGUID := common.NormalizeGUIDUpper(guid)

		// Check if this is the device we're looking for
		// Try exact match first, then partial match for IL-2 compatibility
		if !strings.EqualFold(normalizedGUID, normalizedDeviceGUID) {
			if !common.MatchGUIDPartial(normalizedGUID, normalizedDeviceGUID) {
				continue
			}
		}

		// Process button bindings
		for _, buttonBinding := range device.ButtonBindings {
			for _, buttonID := range buttonBinding.Buttons {
				binding := &Binding{
					DeviceGUID: deviceGUID,
					DeviceName: device.Name,
					InputType:  common.Button,
					InputID:    fmt.Sprintf("%d", buttonID+1), // +1 offset for template compatibility
					Action:     buttonBinding.Action,
				}
				deviceBindings = append(deviceBindings, binding)
			}
		}
	}

	return deviceBindings
}
