// Package il2korea parses IL-2 Sturmovik Korea controller configurations.
//
// Korea uses a JSON configuration format that is incompatible with IL-2 Great Battles
// (see the il2 package): devices are declared in known.devices.json and bindings in
// general.actions, using dev<N>_ references instead of joy<N>_ ones.
//
// Korea ships no human readable action labels, so descriptions are borrowed from a
// configured Great Battles installation when one is available.
package il2korea

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"simdiag/common"
	"simdiag/il2"
)

const (
	knownDevicesFileName   = "known.devices.json"
	generalActionsFileName = "general.actions"
	globalActionsFileName  = "global.actions"
)

// Binding reference patterns. A reference may carry a leading "-" for axis inversion,
// which is stripped before matching (inversion is not represented in diagrams).
var (
	buttonPattern = regexp.MustCompile(`^dev(\d+)_b(\d+)$`)
	axisPattern   = regexp.MustCompile(`^dev(\d+)_axis_([a-z])$`)
	povPattern    = regexp.MustCompile(`^dev(\d+)_pov(\d+)_(\d+)$`)
)

// knownDevicesFile mirrors known.devices.json, which maps a device GUID to its metadata.
// This is the only place the GUID appears; general.actions references devices by numeric ID.
type knownDevicesFile struct {
	KnownDevices map[string]struct {
		DeviceID int    `json:"deviceId"`
		Model    string `json:"model"`
	} `json:"knownDevices"`
}

// generalActionsFile mirrors general.actions. Unknown fields are ignored so that game
// updates adding new keys do not break parsing.
type generalActionsFile struct {
	Devices []struct {
		DeviceID int    `json:"deviceId"`
		Model    string `json:"model"`
	} `json:"devices"`
	Actions map[string][]string `json:"actions"`
}

// Parser implements the SimulatorParser interface for IL-2 Korea.
//
// Unlike the other parsers it holds the configuration, because action labels are read
// from the Great Battles installation declared in it.
type Parser struct {
	config *common.Config
}

// NewParser creates a new IL-2 Korea parser instance.
// config may be nil, in which case action names are used as-is for display.
func NewParser(config *common.Config) *Parser {
	return &Parser{config: config}
}

// GetName implements SimulatorParser.GetName
func (p *Parser) GetName() string {
	return "IL-2 Korea"
}

// Parse implements SimulatorParser.Parse
func (p *Parser) Parse(basePath string) (*common.ProfileCollection, error) {
	knownDevicesPath := filepath.Join(basePath, knownDevicesFileName)
	actionsPath := filepath.Join(basePath, generalActionsFileName)

	// devicesByID resolves a dev<N> reference to a device carrying its GUID
	devicesByID, err := parseKnownDevices(knownDevicesPath)
	if err != nil {
		return nil, err
	}

	actions, roster, err := parseGeneralActions(actionsPath)
	if err != nil {
		return nil, err
	}

	profile := &common.Profile{
		Name:     "IL-2 Korea",
		SimType:  common.IL2Korea,
		Devices:  make(map[string]*common.Device),
		Bindings: make([]common.Binding, 0),
	}

	// The roster lists the devices actually used by this configuration, which may be a
	// subset of the known devices (a device with no binding at all is left out).
	for _, deviceID := range roster {
		if device, exists := devicesByID[deviceID]; exists {
			profile.Devices[device.GUID] = device
		}
	}

	p.addBindings(profile, actions, devicesByID)

	return &common.ProfileCollection{Profiles: []*common.Profile{profile}}, nil
}

// parseKnownDevices reads known.devices.json and returns devices indexed by their numeric ID.
func parseKnownDevices(path string) (map[int]*common.Device, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s not found or unreadable in IL-2 Korea input directory: %w", knownDevicesFileName, err)
	}

	var parsed knownDevicesFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", knownDevicesFileName, err)
	}

	devices := make(map[int]*common.Device, len(parsed.KnownDevices))
	for guid, entry := range parsed.KnownDevices {
		devices[entry.DeviceID] = &common.Device{
			GUID: guid,
			Name: entry.Model,
		}
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no devices found in %s", knownDevicesFileName)
	}

	return devices, nil
}

// parseGeneralActions reads general.actions and returns the action bindings plus the
// list of device IDs in use.
func parseGeneralActions(path string) (map[string][]string, []int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("%s not found or unreadable in IL-2 Korea input directory: %w", generalActionsFileName, err)
	}

	var parsed generalActionsFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, nil, fmt.Errorf("error parsing %s: %w", generalActionsFileName, err)
	}

	roster := make([]int, 0, len(parsed.Devices))
	for _, device := range parsed.Devices {
		roster = append(roster, device.DeviceID)
	}

	return parsed.Actions, roster, nil
}

// loadActionDescriptions borrows the action name -> label mapping from a configured
// IL-2 Great Battles installation. Korea stores no labels of its own, so without Great
// Battles the raw action names are displayed instead.
func (p *Parser) loadActionDescriptions() map[string]string {
	if p.config == nil || p.config.Simulators == nil {
		return map[string]string{}
	}

	gbConfig := p.config.Simulators[common.IL2Sturmovik.GetConfigKey()]
	if gbConfig == nil || gbConfig.IL2InputPath == "" {
		return map[string]string{}
	}

	return il2.LoadActionDescriptions(filepath.Join(gbConfig.IL2InputPath, globalActionsFileName))
}

// addBindings converts every action reference into a Binding on the profile.
func (p *Parser) addBindings(profile *common.Profile, actions map[string][]string, devicesByID map[int]*common.Device) {
	descriptions := p.loadActionDescriptions()

	// Sort action names so that the produced binding order is stable across runs
	actionNames := make([]string, 0, len(actions))
	for actionName := range actions {
		actionNames = append(actionNames, actionName)
	}
	sort.Strings(actionNames)

	for _, actionName := range actionNames {
		description := descriptions[actionName]
		if description == "" {
			description = actionName
		}

		for _, entry := range actions[actionName] {
			// A single entry may hold both directions of a two-way action, separated by "/"
			for _, ref := range strings.Split(entry, "/") {
				ref = strings.TrimSpace(ref)
				// A leading "-" marks an inverted axis, which diagrams do not represent
				ref = strings.TrimPrefix(ref, "-")
				if ref == "" {
					continue
				}

				if binding := p.refToBinding(ref, actionName, description, devicesByID); binding != nil {
					profile.Bindings = append(profile.Bindings, *binding)
				}
			}
		}
	}
}

// refToBinding converts a single binding reference to a Binding, or nil if the reference
// is not usable (mouse input, unknown device, unrecognized format).
func (p *Parser) refToBinding(ref, actionName, description string, devicesByID map[int]*common.Device) *common.Binding {
	// Keyboard bindings are kept for Gremlins/TARGET matching, not for direct display
	if strings.HasPrefix(ref, "key_") {
		return &common.Binding{
			DeviceGUID:  "keyboard",
			DeviceName:  "Keyboard",
			InputType:   common.Button,
			InputID:     ref,
			Action:      actionName,
			Description: description,
		}
	}

	// Mouse bindings are not represented on controller diagrams
	if strings.HasPrefix(ref, "mouse_") {
		return nil
	}

	if matches := buttonPattern.FindStringSubmatch(ref); matches != nil {
		device := lookupDevice(matches[1], devicesByID, ref)
		if device == nil {
			return nil
		}
		// IL-2 button numbers are 0-based, we use 1-based
		buttonNum, err := strconv.Atoi(matches[2])
		if err != nil {
			return nil
		}
		return newDeviceBinding(device, common.Button, strconv.Itoa(buttonNum+1), actionName, description)
	}

	if matches := axisPattern.FindStringSubmatch(ref); matches != nil {
		device := lookupDevice(matches[1], devicesByID, ref)
		if device == nil {
			return nil
		}
		return newDeviceBinding(device, common.Axis, il2.AxisLetterToAxisID(matches[2]), actionName, description)
	}

	if matches := povPattern.FindStringSubmatch(ref); matches != nil {
		device := lookupDevice(matches[1], devicesByID, ref)
		if device == nil {
			return nil
		}
		// IL-2 POV numbers are 0-based, we use 1-based
		povNum, err := strconv.Atoi(matches[2])
		if err != nil {
			return nil
		}
		inputID := fmt.Sprintf("%d_%s", povNum+1, il2.PovAngleToDirection(matches[3]))
		return newDeviceBinding(device, common.Hat, inputID, actionName, description)
	}

	return nil
}

// lookupDevice resolves a dev<N> device number to a known device.
func lookupDevice(deviceNum string, devicesByID map[int]*common.Device, ref string) *common.Device {
	deviceID, err := strconv.Atoi(deviceNum)
	if err != nil {
		return nil
	}

	device, exists := devicesByID[deviceID]
	if !exists {
		fmt.Printf("  [IL2 Korea] Warning: No device found for dev%d (reference '%s')\n", deviceID, ref)
		return nil
	}

	return device
}

// newDeviceBinding builds a Binding for a physical device input.
func newDeviceBinding(device *common.Device, inputType common.InputType, inputID, actionName, description string) *common.Binding {
	return &common.Binding{
		DeviceGUID:  device.GUID,
		DeviceName:  device.Name,
		InputType:   inputType,
		InputID:     inputID,
		Action:      actionName,
		Description: description,
	}
}
