package common

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Regex patterns to extract template keys (case-insensitive)
	buttonPattern   = regexp.MustCompile(`(?i)\bButton_\d+\b`)
	axisPattern     = regexp.MustCompile(`(?i)\bAXIS_[a-zA-Z]+_?\d*\b`)
	hatPattern      = regexp.MustCompile(`(?i)\bPOV_\d+_[URDL]+\b`)
	modifierPattern = regexp.MustCompile(`(?i)\b[a-zA-Z]+_\w+_Modifier_\d+\b`)
)

// GetTemplateNameFromPath extracts template name from filepath (basename without extension)
func GetTemplateNameFromPath(templatePath string) string {
	return filepath.Base(templatePath)
}

// LoadTemplate loads an SVG template file and extracts its keys
func LoadTemplate(templatePath string) (*Template, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("error reading template: %w", err)
	}

	template := &Template{
		FilePath: templatePath,
		Name:     filepath.Base(templatePath),
		RawData:  string(data),
	}

	// Extract template keys
	template.Buttons = extractUniqueMatches(buttonPattern, template.RawData)
	template.Axes = extractUniqueMatches(axisPattern, template.RawData)
	template.Hats = extractUniqueMatches(hatPattern, template.RawData)
	template.Modifiers = extractUniqueMatches(modifierPattern, template.RawData)

	return template, nil
}

// extractUniqueMatches extracts all unique matches from a pattern
func extractUniqueMatches(pattern *regexp.Regexp, text string) []string {
	matches := pattern.FindAllString(text, -1)
	uniqueMap := make(map[string]bool)
	var unique []string

	for _, match := range matches {
		key := strings.ToUpper(match)
		if !uniqueMap[key] {
			uniqueMap[key] = true
			unique = append(unique, key)
		}
	}

	return unique
}

// FindTemplates searches for all .svg files in a directory
func FindTemplates(templatesDir string) ([]*Template, error) {
	var templates []*Template

	err := filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".svg") {
			template, err := LoadTemplate(path)
			if err != nil {
				fmt.Printf("Warning: unable to load %s: %v\n", path, err)
				return nil // Continue with other files
			}
			templates = append(templates, template)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error searching templates: %w", err)
	}

	return templates, nil
}

// CheckCompatibility checks if a device is compatible with a template
func CheckCompatibility(device *Device, profile *Profile, template *Template) (compatible bool, score int, missing []string) {
	deviceInputs := collectDeviceInputs(device, profile)
	score = calculateMatchScore(deviceInputs, template)
	missing = findMissingInputs(deviceInputs, template)
	compatible = isCompatible(score, template)

	return compatible, score, missing
}

// collectDeviceInputs collects all input keys for a device from its bindings
func collectDeviceInputs(device *Device, profile *Profile) map[string]bool {
	deviceInputs := make(map[string]bool)

	for _, binding := range profile.Bindings {
		if binding.DeviceGUID != device.GUID {
			continue
		}

		key := formatInputKey(binding)
		if key != "" {
			deviceInputs[key] = true
		}
	}

	return deviceInputs
}

// formatInputKey formats a binding into a template key format
func formatInputKey(binding Binding) string {
	switch binding.InputType {
	case Button:
		return fmt.Sprintf("BUTTON_%s", binding.InputID)
	case Axis:
		return fmt.Sprintf("AXIS_%s", strings.ToUpper(binding.InputID))
	case Hat:
		return fmt.Sprintf("POV_%s", strings.ToUpper(binding.InputID))
	default:
		return ""
	}
}

// calculateMatchScore counts how many device inputs match template inputs
func calculateMatchScore(deviceInputs map[string]bool, template *Template) int {
	score := 0

	for _, templateButton := range template.Buttons {
		if deviceInputs[templateButton] {
			score++
		}
	}

	for _, templateAxis := range template.Axes {
		if deviceInputs[templateAxis] {
			score++
		}
	}

	for _, templateHat := range template.Hats {
		if deviceInputs[templateHat] {
			score++
		}
	}

	return score
}

// findMissingInputs finds device inputs that are not in the template
func findMissingInputs(deviceInputs map[string]bool, template *Template) []string {
	missing := make([]string, 0)

	for inputKey := range deviceInputs {
		if !isInputInTemplate(inputKey, template) {
			missing = append(missing, inputKey)
		}
	}

	return missing
}

// isInputInTemplate checks if an input key exists in the template
func isInputInTemplate(inputKey string, template *Template) bool {
	for _, templateKey := range template.Buttons {
		if templateKey == inputKey {
			return true
		}
	}

	for _, templateKey := range template.Axes {
		if templateKey == inputKey {
			return true
		}
	}

	for _, templateKey := range template.Hats {
		if templateKey == inputKey {
			return true
		}
	}

	return false
}

// isCompatible determines if the device is compatible based on match score
func isCompatible(score int, template *Template) bool {
	totalTemplateKeys := len(template.Buttons) + len(template.Axes) + len(template.Hats)
	if totalTemplateKeys == 0 {
		return true
	}

	compatibilityRatio := float64(score) / float64(totalTemplateKeys)
	return compatibilityRatio >= 0.5
}
