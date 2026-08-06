package svg

import (
	"cmp"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"simdiag/common"
)

// Template metadata placeholders, substituted once per generated diagram.
var (
	templateNamePattern = regexp.MustCompile(`\bTEMPLATE_NAME\b`)
	currentDatePattern  = regexp.MustCompile(`\bCURRENT_DATE\b`)
	dateTimePattern     = regexp.MustCompile(`\bDATE_TIME\b`)
	outputDirPattern    = regexp.MustCompile(`\bOUTPUT_DIRECTORY\b`)
	simulatorPattern    = regexp.MustCompile(`\bSIMULATOR\b`)
	versionPattern      = regexp.MustCompile(`\bSIMDIAG_VERSION\b`)
	titlePattern        = regexp.MustCompile(`\bTITLE\b`)
)

// unusedKeyPatterns match the placeholders that must be blanked out when no
// binding filled them in: buttons, axes, hats and the modifier colour slots.
var unusedKeyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bButton_\d+\b`),
	regexp.MustCompile(`\bAxis_[a-zA-Z]+_?\d*\b`),
	regexp.MustCompile(`\bPOV_\d+_[URDL]+\b`),
	regexp.MustCompile(`\b[a-zA-Z]+_\w+_Modifiers?\b`),
	regexp.MustCompile(`\b[a-zA-Z]+_\w+_Modifier_\d+(_[a-zA-Z]+)?\b`),
}

// ExportToSVG exports a device to an SVG file using a template
func ExportToSVG(exportDevice *common.ExportDevice, outputDir string) error {
	if exportDevice.Template == nil {
		return fmt.Errorf("template missing for export")
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Fill template with device data
	svgData := populateTemplate(exportDevice)

	// Note: Pretty-printing SVG with Go's XML encoder causes issues with DOCTYPE and namespaces
	// Keep the original formatting from the template

	// Generate file name based on template name
	outputPath := filepath.Join(outputDir, filepath.Base(exportDevice.Template.FilePath))

	if err := os.WriteFile(outputPath, []byte(svgData), 0644); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	fmt.Printf("✓ Diagram exported: %s\n", outputPath)

	// Convert the diagram to PNG in a png subdirectory; failures are reported but
	// do not fail the export, since the SVG is already written.
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

		key := common.TemplateKeyFor(binding.InputType, binding.InputID)

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
		action := binding.DisplayText()

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
		modifierAction := binding.DisplayText()
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

// replaceKeys replaces template keys with actions (with multi-binding support).
// Orchestrates buildHTMLForBindings and replaceKeyInTemplate.
func replaceKeys(data string, bindingMap map[string][]common.Binding) string {
	// Sort keys by length (descending) to avoid partial replacements, then by name
	// so that same-length keys are always processed in the same order
	keys := make([]string, 0, len(bindingMap))
	for key := range bindingMap {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int {
		return cmp.Or(cmp.Compare(len(b), len(a)), cmp.Compare(a, b))
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
	now := time.Now()

	// Placeholders that are always substituted
	for _, sub := range []struct {
		pattern *regexp.Regexp
		value   string
	}{
		{templateNamePattern, sanitizeForSVG(exportDevice.Profile.Name)},
		{currentDatePattern, now.Format("2006-01-02")},
		{dateTimePattern, now.Format("2006-01-02 15:04:05")},
	} {
		data = sub.pattern.ReplaceAllString(data, sub.value)
	}

	// Placeholders left untouched when the export device carries no value, so the
	// template can fall back to whatever it already displays
	for _, sub := range []struct {
		pattern *regexp.Regexp
		value   string
	}{
		{outputDirPattern, exportDevice.OutputDirectory},
		{simulatorPattern, exportDevice.SimulatorName},
		{versionPattern, exportDevice.SimdiagVersion},
		{titlePattern, exportDevice.Title},
	} {
		if sub.value != "" {
			data = sub.pattern.ReplaceAllString(data, sub.value)
		}
	}

	return data
}

// replaceUnusedKeys blanks out the template placeholders that no binding filled in
func replaceUnusedKeys(data string) string {
	for _, pattern := range unusedKeyPatterns {
		data = pattern.ReplaceAllString(data, "")
	}
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

// modifierColors holds the colour of modifier N at index N-1.
var modifierColors = [...]string{
	"rgb(0, 153, 0)",     // 1  Green
	"rgb(234, 107, 102)", // 2  Salmon
	"rgb(255, 153, 0)",   // 3  Orange
	"rgb(255, 1, 1)",     // 4  Red
	"rgb(255, 1, 242)",   // 5  Magenta
	"rgb(255, 215, 0)",   // 6  Gold/Yellow
	"rgb(148, 0, 211)",   // 7  Dark Violet
	"rgb(255, 105, 180)", // 8  Hot Pink
	"rgb(255, 69, 0)",    // 9  Orange Red
	"rgb(50, 205, 50)",   // 10 Lime Green
}

// getModifierColor returns RGB color for a modifier number
func getModifierColor(modNum int) string {
	if modNum >= 1 && modNum <= len(modifierColors) {
		return modifierColors[modNum-1]
	}
	return modifierColors[0] // Default to green
}

// sanitizeForSVG escapes special characters for SVG display
func sanitizeForSVG(text string) string {
	// First normalize characters (replace & with /, other special chars with _)
	text = normalizeText(text)
	// Then escape remaining XML/HTML characters
	text = html.EscapeString(text)
	return text
}
