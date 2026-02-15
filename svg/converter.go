package svg

import (
	"fmt"
	"os"
	"os/exec"
	"simdiag/common"
)

// ConvertSVGToPNG converts an SVG file to PNG format using Draw.io
// Requires Draw.io to be installed and available in PATH
func ConvertSVGToPNG(svgPath, pngPath string, config *common.Config) error {
	var drawIOCmd string

	// First, try configured path from config
	if config != nil && config.DrawIOPath != "" {
		if _, err := os.Stat(config.DrawIOPath); err == nil {
			drawIOCmd = config.DrawIOPath
		}
	}

	// If not found in config, try default paths
	if drawIOCmd == "" {
		drawIOPaths := []string{
			"draw.io", // In PATH
			"C:\\Program Files\\draw.io\\draw.io.exe", // Default Windows installation
		}

		for _, path := range drawIOPaths {
			if _, err := exec.LookPath(path); err == nil {
				drawIOCmd = path
				break
			}
			// On Windows, check if file exists directly
			if _, err := os.Stat(path); err == nil {
				drawIOCmd = path
				break
			}
		}
	}

	if drawIOCmd == "" {
		return fmt.Errorf("draw.io not found. Please install Draw.io from https://github.com/jgraph/drawio-desktop/releases")
	}

	// Use Draw.io for conversion
	// Syntax: draw.io.exe --export --format png --output "foo.png" "bar.svg"
	cmd := exec.Command(drawIOCmd, "--export", "--format", "png", "--output", pngPath, svgPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("draw.io conversion failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
