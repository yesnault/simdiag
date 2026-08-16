package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"simdiag/app"
	"simdiag/common"
	"simdiag/gui"
	"simdiag/svg"
	"simdiag/update"
	"simdiag/workflow"
)

// simdiagVersion is set at build time via ldflags
var simdiagVersion = "dev"

func init() {
	// Set the version in common package so it's available for exports
	common.SimdiagVersion = simdiagVersion
}

func main() {
	// No arguments at all means a double-click: open the graphical interface.
	// Every other invocation is a command line one and needs a console.
	if len(os.Args) == 1 {
		if err := gui.Run(simdiagVersion); err != nil {
			app.AttachParentConsole()
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// SimDiag is linked as a GUI-subsystem binary, so a CLI run has no standard
	// handles until it reattaches to the console that launched it.
	app.AttachParentConsole()

	// Handle 'update' subcommand before flag parsing
	if len(os.Args) >= 2 && os.Args[1] == "update" {
		if err := update.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	args := parseFlags()
	common.SetConfigFileName(args.configFile)

	if args.versionFlag {
		fmt.Printf("SimDiag version %s\n", simdiagVersion)
		os.Exit(0)
	}

	if err := validateFlags(args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("=== SimDiag - Simulator Diagram Generator ===")
	fmt.Println()

	if err := runApplication(args); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}
}

// cliArgs holds parsed command line arguments
type cliArgs struct {
	batchMode   bool
	versionFlag bool
	configFile  string
	csvPath     string
	filter      string
	noSVG       bool
}

// parseFlags parses command line flags and returns structured arguments.
//
// The command line exports; it does not configure. Creating and editing
// mapping_config.yaml is the graphical interface's job. No flag here writes to
// it.
func parseFlags() cliArgs {
	args := cliArgs{}
	flag.BoolVar(&args.batchMode, "b", false, "Batch mode: export everything the configuration declares")
	flag.BoolVar(&args.versionFlag, "v", false, "Display version information")
	flag.StringVar(&args.configFile, "c", "mapping_config.yaml", "Path to the configuration file")
	flag.StringVar(&args.csvPath, "csv", "", "Generate SVG/PNG from existing CSV file")
	flag.StringVar(&args.filter, "f", "", "Filter modules/simulators in batch mode (e.g., '2000' for M-2000C, 'il-2' for IL-2)")
	flag.BoolVar(&args.noSVG, "no-svg", false, "CSV export only (skip SVG generation)")
	flag.Parse()
	return args
}

// validateFlags checks for conflicting flag combinations
func validateFlags(args cliArgs) error {
	if args.batchMode && args.csvPath != "" {
		return fmt.Errorf("❌ Error: Cannot use -b and -csv flags together\nUse -b for full batch processing or -csv to generate from existing CSV")
	}
	return nil
}

// runApplication executes the appropriate workflow based on flags
func runApplication(args cliArgs) error {
	if args.csvPath != "" {
		return runCSVMode(args.csvPath, args.configFile, args.noSVG)
	}

	// Flags that select no export mode (typically just -c) mean the graphical
	// interface, on the requested configuration.
	if !args.batchMode {
		return gui.RunWithConfig(simdiagVersion, args.configFile)
	}

	config, err := loadConfig()
	if err != nil {
		return err
	}

	runBatchMode(config, args.filter, args.noSVG)
	return nil
}

// openTheInterface is the way out of every "nothing is configured" error. The
// command line cannot create a configuration any more. Pointing at a flag
// would be pointing at nothing.
const openTheInterface = "Run simdiag.exe with no argument to create one in the graphical interface."

// loadConfig loads the configuration an export needs. It has to exist: the
// command line reads the configuration, it never writes one.
func loadConfig() (*common.Config, error) {
	if _, err := os.Stat(common.GetConfigFileName()); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s\n%s", common.GetConfigFileName(), openTheInterface)
	}

	config, err := common.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to load configuration file: %s\nDetails: %w\n%s", common.GetConfigFileName(), err, openTheInterface)
	}
	if config == nil {
		return nil, fmt.Errorf("unable to load configuration file: %s\n%s", common.GetConfigFileName(), openTheInterface)
	}

	if err := validateBatchConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// validateBatchConfig ensures the configuration is valid for batch mode.
//
// It asks common.SimulatorIsConfigured the same question the export itself asks,
// rather than re-stating the rule: this check used to test the DCS module list
// and never DCSPath. A DCS-only configuration was refused here even though the
// export would have handled it.
func validateBatchConfig(config *common.Config) error {
	hasConfiguredSim := false
	for key, simConfig := range config.Simulators {
		if common.SimulatorIsConfigured(common.SimulationTypeForConfigKey(key), simConfig) {
			hasConfiguredSim = true
			break
		}
	}

	if !hasConfiguredSim {
		return fmt.Errorf("no configured simulators found.\nThe configuration file exists but declares no simulator path.\n%s", openTheInterface)
	}
	return nil
}

// runCSVMode generates SVG/PNG from an existing CSV file
func runCSVMode(csvPath, configFile string, noSVG bool) error {
	// Verify CSV file exists
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		return fmt.Errorf("CSV file not found: %s", csvPath)
	}

	// Verify config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s\nConfiguration is required to locate templates and output directories", configFile)
	}

	// Load config
	config, err := common.LoadConfig()
	if err != nil {
		return fmt.Errorf("unable to load configuration file: %s\nDetails: %w", configFile, err)
	}
	if config == nil {
		return fmt.Errorf("unable to load configuration file: %s", configFile)
	}

	// Validate config has required fields
	if config.TemplatesDirectory == "" {
		// The base templates ship inside this binary; the graphical interface
		// offers to write them out, which is the shortest way out of this.
		return fmt.Errorf("templates_directory not configured: run simdiag.exe with no argument to install the base templates")
	}
	if config.OutputDirectory == "" {
		return fmt.Errorf("output_directory not configured")
	}

	if noSVG {
		fmt.Printf("CSV mode with --no-svg flag: CSV file already exists at %s\n", csvPath)
		return nil
	}

	// Check draw.io availability (optional - only affects PNG export)
	if _, found := common.VerifyDrawIOPath(config); !found {
		fmt.Println("⚠ Warning: draw.io not found. PNG export will be skipped (SVG only).")
	}

	fmt.Printf("Generating SVG/PNG from CSV: %s\n", csvPath)

	// Generate SVG from CSV. Validation errors are already printed by the
	// generator. The CLI has nothing extra to do with them.
	if _, err := svg.GenerateSVGFromCSV(context.Background(), csvPath, config); err != nil {
		return err
	}

	fmt.Println("\n✓ Diagrams generated successfully")
	return nil
}

// runBatchMode executes the batch export workflow
func runBatchMode(config *common.Config, filter string, noSVG bool) {
	if noSVG {
		fmt.Println("Running in CSV-only mode (SVG generation skipped)")
	} else {
		// Check draw.io availability (optional - only affects PNG export)
		if _, found := common.VerifyDrawIOPath(config); !found {
			fmt.Println("⚠ Warning: draw.io not found. PNG export will be skipped (SVG only).")
			fmt.Println("  To enable PNG export, install draw.io and configure its path in the config file.")
		}
	}

	// Process all configured simulators, reusing the config already loaded
	if _, err := workflow.ExportAll(context.Background(), config, app.Parsers(config), app.Enrichers(), filter, noSVG); err != nil {
		fmt.Printf("⚠ Export failed: %v\n", err)
	}
}
