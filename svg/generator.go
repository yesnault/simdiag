package svg

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"simdiag/common"
	"sort"
	"strings"
	"time"
)

// ExportToSVG exports a device to an SVG file using a template
func ExportToSVG(exportDevice *common.ExportDevice, outputDir string) error {
	if exportDevice.Template == nil {
		return fmt.Errorf("template missing for export")
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Fill template with device data
	svgData := populateTemplate(exportDevice)

	// Note: Pretty-printing SVG with Go's XML encoder causes issues with DOCTYPE and namespaces
	// Keep the original formatting from the template

	// Generate file name based on template name
	fileName := filepath.Base(exportDevice.Template.FilePath)

	outputPath := filepath.Join(outputDir, fileName)

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Save file
	if err := os.WriteFile(outputPath, []byte(svgData), 0644); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	fmt.Printf("✓ Diagram exported: %s\n", outputPath)

	// Convert to PNG if requested

	// Create png subdirectory
	pngDir := filepath.Join(outputDir, "png")
	if err := os.MkdirAll(pngDir, 0755); err != nil {
		fmt.Printf("  ⚠ PNG directory creation failed: %v\n", err)
	} else {
		// Generate PNG filename in png subdirectory
		baseName := filepath.Base(outputPath)
		pngFileName := strings.TrimSuffix(baseName, ".svg") + ".png"
		pngPath := filepath.Join(pngDir, pngFileName)

		if err := ConvertSVGToPNG(outputPath, pngPath, nil); err != nil {
			fmt.Printf("  ⚠ PNG conversion failed: %v\n", err)
		} else {
			fmt.Printf("  ✓ PNG exported: %s\n", pngPath)
		}
	}

	return nil
}

// populateTemplate replaces template keys with device values
// getBindingDisplayText returns the best display text for a binding
// For IL-2: use Description (e.g. "Suralimentation") if available
// For DCS/SRS: use Action
func getBindingDisplayText(binding common.Binding) string {
	if binding.Description != "" {
		return binding.Description
	}
	return binding.Action
}

func populateTemplate(exportDevice *common.ExportDevice) string {
	data := exportDevice.Template.RawData

	// Create a binding map for fast access and a map for modifiers
	// Map to store ALL actions by key (supports multiple bindings per button)
	bindingMap := make(map[string][]common.Binding)

	// Build a map of all device GUIDs in this export (for merged devices)
	deviceGUIDs := make(map[string]bool)
	for guid := range exportDevice.Profile.Devices {
		deviceGUIDs[common.NormalizeGUIDShort(guid)] = true
	}

	for _, binding := range exportDevice.Profile.Bindings {
		bindingID := common.NormalizeGUIDShort(binding.DeviceGUID)

		// Check if this binding belongs to ANY of the merged devices
		if !deviceGUIDs[bindingID] {
			continue
		}

		// Strip _OFF suffix so BTN25_OFF maps to Button_25 (not Button_25_OFF)
		templateInputID := strings.TrimSuffix(binding.InputID, "_OFF")

		var key string
		switch binding.InputType {
		case common.Button:
			key = fmt.Sprintf("Button_%s", templateInputID)
		case common.Axis:
			key = fmt.Sprintf("AXIS_%s", strings.ToUpper(templateInputID))
		case common.Hat:
			key = fmt.Sprintf("POV_%s", strings.ToUpper(templateInputID))
		}

		// ACCUMULATE all actions instead of overwriting
		bindingMap[key] = append(bindingMap[key], binding)
	}

	// Replace keys in template (now handles modifiers too)
	data = replaceKeys(data, bindingMap)

	// Replace metadata
	data = replaceMetadata(data, exportDevice)

	// Replace unused keys with empty values
	data = replaceUnusedKeys(data)

	return data
}

// isDuplicateHTML checks if text already exists in any of the htmlParts.
func isDuplicateHTML(htmlParts []string, text string) bool {
	for _, existing := range htmlParts {
		if strings.Contains(existing, text) {
			return true
		}
	}
	return false
}

// buildHTMLForBindings generates HTML parts for bindings on a single key.
// Returns the list of HTML div elements for the combined binding display.
func buildHTMLForBindings(bindingsWithoutModifier, bindingsWithModifier []common.Binding) []string {
	var htmlParts []string

	// Process actions without modifiers
	for _, binding := range bindingsWithoutModifier {
		action := getBindingDisplayText(binding)

		if action == "" {
			continue
		}

		actionLines := strings.Split(action, "\n")
		for _, actionLine := range actionLines {
			actionLine = strings.TrimSpace(actionLine)
			if actionLine == "" || isDuplicateHTML(htmlParts, actionLine) {
				continue
			}

			// If binding has a modifier number, apply color
			if binding.ModifierNum > 0 {
				color := getModifierColor(binding.ModifierNum)
				htmlParts = append(htmlParts, fmt.Sprintf(`<div><font style="color: %s;">%s</font></div>`, color, actionLine))
			} else {
				htmlParts = append(htmlParts, fmt.Sprintf("<div>%s</div>", actionLine))
			}
		}
	}

	// Process actions with modifiers
	for _, binding := range bindingsWithModifier {
		modifierAction := getBindingDisplayText(binding)
		if modifierAction == "" {
			continue
		}

		// Use modifier number from binding
		modNum := binding.ModifierNum
		if modNum == 0 {
			modNum = 1 // Default color
		}
		color := getModifierColor(modNum)

		if !isDuplicateHTML(htmlParts, modifierAction) {
			htmlParts = append(htmlParts, fmt.Sprintf(`<div><font style="color: %s;">%s</font></div>`, color, modifierAction))
		}
	}

	return htmlParts
}

// replaceKeyInTemplate replaces a template key (e.g., Button_3) with escaped HTML in the SVG data.
func replaceKeyInTemplate(data, key, escapedHTML string) string {
	if escapedHTML == "" {
		return data
	}

	// Try multiple patterns in order of specificity
	// Pattern 1: Button_X&amp;lt;br /&amp;gt; or Button_X&amp;lt;br/&amp;gt;
	brPattern := regexp.MustCompile(regexp.QuoteMeta(key) + `&amp;lt;br\s*/&amp;gt;`)
	if brPattern.MatchString(data) {
		return brPattern.ReplaceAllString(data, escapedHTML)
	}

	// Pattern 2: &amp;lt;div&amp;gt;Button_X&amp;lt;/div&amp;gt;
	divPattern := regexp.MustCompile(`&amp;lt;div&amp;gt;\s*` + regexp.QuoteMeta(key) + `\s*&amp;lt;/div&amp;gt;`)
	if divPattern.MatchString(data) {
		return divPattern.ReplaceAllString(data, escapedHTML)
	}

	// Pattern 3: Simple word boundary replacement
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(key) + `\b`)
	return pattern.ReplaceAllString(data, escapedHTML)
}

// replaceBackwardCompatModifiers replaces legacy Button_X_Modifier_N and Button_X_Modifiers patterns.

// replaceKeys replaces template keys with actions (with multi-binding support).
// Orchestrates buildHTMLForBindings and replaceKeyInTemplate.
func replaceKeys(data string, bindingMap map[string][]common.Binding) string {
	// Sort keys by length (descending) to avoid partial replacements
	keys := make([]string, 0, len(bindingMap))
	for key := range bindingMap {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	for _, key := range keys {
		bindings := bindingMap[key]
		if len(bindings) == 0 {
			continue
		}

		// Separate bindings with and without modifiers
		var bindingsWithoutModifier, bindingsWithModifier []common.Binding
		for _, binding := range bindings {
			if len(binding.Modifiers) == 0 {
				bindingsWithoutModifier = append(bindingsWithoutModifier, binding)
			} else {
				bindingsWithModifier = append(bindingsWithModifier, binding)
			}
		}

		// Build HTML content
		htmlParts := buildHTMLForBindings(bindingsWithoutModifier, bindingsWithModifier)

		// Escape for SVG (double escaping for XML attributes)
		combinedHTML := strings.Join(htmlParts, "")
		escapedHTML := html.EscapeString(combinedHTML)
		escapedHTML = strings.ReplaceAll(escapedHTML, "&#34;", "&quot;")
		escapedHTML = strings.ReplaceAll(escapedHTML, "&", "&amp;")

		// Replace key in template
		data = replaceKeyInTemplate(data, key, escapedHTML)
	}

	return data
}

// replaceMetadata replaces template metadata
func replaceMetadata(data string, exportDevice *common.ExportDevice) string {
	// Remplacer TEMPLATE_NAME
	templateNamePattern := regexp.MustCompile(`\bTEMPLATE_NAME\b`)
	profileName := sanitizeForSVG(exportDevice.Profile.Name)
	data = templateNamePattern.ReplaceAllString(data, profileName)

	// Remplacer CURRENT_DATE
	currentDatePattern := regexp.MustCompile(`\bCURRENT_DATE\b`)
	currentDate := time.Now().Format("2006-01-02")
	data = currentDatePattern.ReplaceAllString(data, currentDate)

	// Remplacer DATE_TIME
	dateTimePattern := regexp.MustCompile(`\bDATE_TIME\b`)
	dateTime := time.Now().Format("2006-01-02 15:04:05")
	data = dateTimePattern.ReplaceAllString(data, dateTime)

	// Remplacer OUTPUT_DIRECTORY
	if exportDevice.OutputDirectory != "" {
		outputDirPattern := regexp.MustCompile(`\bOUTPUT_DIRECTORY\b`)
		data = outputDirPattern.ReplaceAllString(data, exportDevice.OutputDirectory)
	}

	// Remplacer SIMULATOR
	if exportDevice.SimulatorName != "" {
		simulatorPattern := regexp.MustCompile(`\bSIMULATOR\b`)
		data = simulatorPattern.ReplaceAllString(data, exportDevice.SimulatorName)
	}

	// Remplacer SIMDIAG_VERSION
	if exportDevice.SimdiagVersion != "" {
		versionPattern := regexp.MustCompile(`\bSIMDIAG_VERSION\b`)
		data = versionPattern.ReplaceAllString(data, exportDevice.SimdiagVersion)
	}

	// Remplacer TITLE
	if exportDevice.Title != "" {
		titlePattern := regexp.MustCompile(`\bTITLE\b`)
		data = titlePattern.ReplaceAllString(data, exportDevice.Title)
	}

	return data
}

// replaceUnusedKeys replaces unused keys with empty strings
func replaceUnusedKeys(data string) string {
	// Replace unused buttons (format: Button_1, Button_99, etc.)
	buttonPattern := regexp.MustCompile(`\bButton_\d+\b`)
	data = buttonPattern.ReplaceAllString(data, "")

	// Replace unused axes (format: Axis_X, Axis_Y, Axis_Z, etc.)
	axisPattern := regexp.MustCompile(`\bAxis_[a-zA-Z]+_?\d*\b`)
	data = axisPattern.ReplaceAllString(data, "")

	// Replace unused hats
	hatPattern := regexp.MustCompile(`\bPOV_\d+_[URDL]+\b`)
	data = hatPattern.ReplaceAllString(data, "")

	// Replace unused modifiers
	modifierPattern := regexp.MustCompile(`\b[a-zA-Z]+_\w+_Modifiers?\b`)
	data = modifierPattern.ReplaceAllString(data, "")

	modifierKeyPattern := regexp.MustCompile(`\b[a-zA-Z]+_\w+_Modifier_\d+(_[a-zA-Z]+)?\b`)
	data = modifierKeyPattern.ReplaceAllString(data, "")

	return data
}

// normalizeText filters text to keep only safe characters for SVG display
// Keeps: a-zA-Z0-9, -, _, space, and accented letters (À-ÿ)
// Replaces '&' with '/' and all other special characters with '_'
func normalizeText(s string) string {
	var result strings.Builder
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' ||
			r == '_' ||
			r == ' ' ||
			(r >= '\u00c0' && r <= '\u00ff'): // Accented characters À-ÿ
			result.WriteRune(r)
		case r == '&':
			result.WriteRune('/') // Replace & with / for better SVG display
		default:
			result.WriteRune('_')
		}
	}
	return result.String()
}

// sanitizeForSVG escapes special characters for SVG display
// getModifierColor returns RGB color for a modifier number
func getModifierColor(modNum int) string {
	colors := map[int]string{
		1:  "rgb(0, 153, 0)",     // Green
		2:  "rgb(234, 107, 102)", // Salmon
		3:  "rgb(255, 153, 0)",   // Orange
		4:  "rgb(255, 1, 1)",     // Red
		5:  "rgb(255, 1, 242)",   // Magenta
		6:  "rgb(255, 215, 0)",   // Gold/Yellow
		7:  "rgb(148, 0, 211)",   // Dark Violet
		8:  "rgb(255, 105, 180)", // Hot Pink
		9:  "rgb(255, 69, 0)",    // Orange Red
		10: "rgb(50, 205, 50)",   // Lime Green
	}
	if color, exists := colors[modNum]; exists {
		return color
	}
	return "rgb(0, 153, 0)" // Default to green
}

func sanitizeForSVG(text string) string {
	// First normalize characters (replace & with /, other special chars with _)
	text = normalizeText(text)
	// Then escape remaining XML/HTML characters
	text = html.EscapeString(text)
	return text
}
