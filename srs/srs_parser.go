package srs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"simdiag/common"
	"strconv"
	"strings"
)

// ParseSRSConfig parses an SRS default.cfg file and returns bindings by device GUID
func ParseSRSConfig(srsPath string, simType common.SimulationType) (map[string][]common.Binding, error) {
	// Construct the path to default.cfg
	var configPath string
	if simType == common.DCSWorld {
		configPath = filepath.Join(srsPath, "Client", "default.cfg")
	} else {
		configPath = filepath.Join(srsPath, "default.cfg")
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("SRS config file not found: %s", configPath)
	}

	// Open and read the file
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("error opening SRS config: %w", err)
	}
	defer file.Close()

	// Parse the INI-style config
	bindings := make(map[string][]common.Binding)
	scanner := bufio.NewScanner(file)

	var currentSection string
	var currentName string
	var currentButton string
	var currentGUID string

	sectionRegex := regexp.MustCompile(`^\[(.+)\]$`)
	nameRegex := regexp.MustCompile(`^name="(.+)"$`)
	buttonRegex := regexp.MustCompile(`^button=(\d+)$`)
	guidRegex := regexp.MustCompile(`^guid=(.+)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Check for section header
		if matches := sectionRegex.FindStringSubmatch(line); matches != nil {
			// Save previous section if complete
			if currentSection != "" && currentGUID != "" && currentButton != "" {
				normalizedGUID := strings.ToLower(strings.TrimSpace(currentGUID))
				binding := common.Binding{
					DeviceGUID:  normalizedGUID,
					DeviceName:  currentName,
					InputType:   common.Button,
					InputID:     currentButton,
					Action:      fmt.Sprintf("SRS: %s", currentSection),
					Description: fmt.Sprintf("SRS %s", currentSection),
				}
				bindings[normalizedGUID] = append(bindings[normalizedGUID], binding)
			}

			// Start new section
			currentSection = matches[1]
			currentName = ""
			currentButton = ""
			currentGUID = ""
			continue
		}

		// Parse properties
		if matches := nameRegex.FindStringSubmatch(line); matches != nil {
			currentName = matches[1]
		} else if matches := buttonRegex.FindStringSubmatch(line); matches != nil {
			// SRS uses 0-indexed buttons, templates use 1-indexed (button=16 in SRS = Button_17 in template)
			buttonNum, err := strconv.Atoi(matches[1])
			if err == nil {
				currentButton = strconv.Itoa(buttonNum + 1)
			} else {
				currentButton = matches[1]
			}
		} else if matches := guidRegex.FindStringSubmatch(line); matches != nil {
			currentGUID = matches[1]
		}
	}

	// Save last section if complete
	if currentSection != "" && currentGUID != "" && currentButton != "" {
		normalizedGUID := strings.ToLower(strings.TrimSpace(currentGUID))
		binding := common.Binding{
			DeviceGUID:  normalizedGUID,
			DeviceName:  currentName,
			InputType:   common.Button,
			InputID:     currentButton,
			Action:      fmt.Sprintf("SRS: %s", currentSection),
			Description: fmt.Sprintf("SRS %s", currentSection),
		}
		bindings[normalizedGUID] = append(bindings[normalizedGUID], binding)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading SRS config: %w", err)
	}

	return bindings, nil
}
