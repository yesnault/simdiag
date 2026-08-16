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

// KeyPatterns returns the patterns that recognise a template placeholder.
//
// Whoever blanks the placeholders no binding filled in must use these and not a
// second list of its own. There used to be one in svg/generator.go, spelled
// Axis_ and case-sensitive where this one is AXIS_ and not; since the shipped
// templates use AXIS_X, no axis placeholder was ever blanked and every unbound
// axis kept its raw key on the finished diagram.
func KeyPatterns() []*regexp.Regexp {
	return []*regexp.Regexp{buttonPattern, axisPattern, hatPattern}
}

// HasTemplateKey reports whether text holds a placeholder the loader would pick
// up. It exists so a test can state the invariant the two lists must satisfy.
func HasTemplateKey(text string) bool {
	for _, pattern := range KeyPatterns() {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

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
				Printf("Warning: unable to load %s: %v\n", path, err)
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
