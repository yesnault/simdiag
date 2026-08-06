package main

import (
	"flag"
	"fmt"
	"os"

	"simdiag/common"
	"simdiag/dcs"
	"simdiag/gremlins"
	"simdiag/il2"
	"simdiag/il2korea"
	"simdiag/openkneeboard"
	"simdiag/srs"
	"simdiag/svg"
	"simdiag/target"
	"simdiag/update"
	"simdiag/workflow"
)

// simdiagVersion is set at build time via ldflags
var simdiagVersion = "dev"

func init() {
	// Set the version in common package so it's available for exports
	common.SimdiagVersion = simdiagVersion

	// Set up external function dependencies
	common.ExtFuncs = &common.ExternalFuncs{
		GetTargetDeviceNumbers:    target.GetTargetDeviceNumbers,
		AutoMatchTargetDevices:    target.AutoMatchTargetDevices,
		TargetDeviceNumberToName:  target.DeviceNumberToName,
		GetUnmatchedTargetDevices: target.GetUnmatchedTargetDevices,
		LoadGremlinsBindingsForDevice: func(guid string, config *common.Config) interface{} {
			return gremlins.LoadBindingsForDevice(guid, config)
		},
		LoadOpenKneeboardBindingsForDevice: func(guid string, config *common.Config) interface{} {
			return openkneeboard.LoadBindingsForDevice(guid, config)
		},
		ParseGremlinsProfile: func(path string) (interface{}, error) { return gremlins.ParseProfile(path) },
	}
}

func main() {
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

// parseFlags parses command line flags and returns structured arguments
func parseFlags() cliArgs {
	args := cliArgs{}
	flag.BoolVar(&args.batchMode, "b", false, "Batch mode: non-interactive, use only existing mappings")
	flag.BoolVar(&args.versionFlag, "v", false, "Display version information")
	flag.StringVar(&args.configFile, "c", "mapping_config.yaml", "Path to the configuration file")
	flag.StringVar(&args.csvPath, "csv", "", "Generate SVG/PNG from existing CSV file")
	flag.StringVar(&args.filter, "f", "", "Filter modules/simulators in batch mode (e.g., '2000' for M-2000C, 'il2' for IL-2)")
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

	config, err := loadConfig(args.batchMode)
	if err != nil {
		return err
	}

	if args.batchMode {
		runBatchMode(config, args.filter, args.noSVG)
		return nil
	}

	return runInteractiveMode(config)
}

// loadConfig loads the configuration file or creates a new one for interactive mode
func loadConfig(batchMode bool) (*common.Config, error) {
	// In batch mode, configuration file is required
	if batchMode {
		if _, err := os.Stat(common.GetConfigFileName()); os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration file not found: %s\nBatch mode requires an existing configuration file.\nPlease run in interactive mode first to create the configuration", common.GetConfigFileName())
		}
	}

	// Load config
	config, err := common.LoadConfig()

	// If config failed to load, handle based on mode
	if err != nil || config == nil {
		if batchMode {
			if err != nil {
				return nil, fmt.Errorf("unable to load configuration file: %s\nDetails: %v\nPlease run in interactive mode first to create a valid configuration", common.GetConfigFileName(), err)
			}
			return nil, fmt.Errorf("unable to load configuration file: %s\nPlease run in interactive mode first to create a valid configuration", common.GetConfigFileName())
		}
		// Interactive mode - create empty config
		config = &common.Config{Simulators: make(map[string]*common.SimulatorConfig)}
		return config, nil
	}

	// In batch mode, validate that config has at least one simulator configured
	if batchMode {
		if err := validateBatchConfig(config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// validateBatchConfig ensures the configuration is valid for batch mode
func validateBatchConfig(config *common.Config) error {
	hasConfiguredSim := false
	for _, simConfig := range config.Simulators {
		if simConfig != nil {
			// Check if DCS has modules or if IL-2 has input path configured
			if len(simConfig.Modules) > 0 || simConfig.IL2InputPath != "" {
				hasConfiguredSim = true
				break
			}
		}
	}

	if !hasConfiguredSim {
		return fmt.Errorf("no configured simulators found.\nThe configuration file exists but contains no simulator configuration.\nPlease run in interactive mode first to configure at least one simulator")
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
		return fmt.Errorf("unable to load configuration file: %s\nDetails: %v", configFile, err)
	}
	if config == nil {
		return fmt.Errorf("unable to load configuration file: %s", configFile)
	}

	// Validate config has required fields
	if config.TemplatesDirectory == "" {
		return fmt.Errorf("templates_directory not configured")
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

	// Generate SVG from CSV
	if err := svg.GenerateSVGFromCSV(csvPath, config); err != nil {
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

	// Create simulator parsers
	parsers := map[common.SimulationType]common.SimulatorParser{
		common.DCSWorld:     dcs.NewParser(),
		common.IL2Sturmovik: il2.NewParser(),
		common.IL2Korea:     il2korea.NewParser(config),
	}

	// Create binding enrichers
	enrichers := []common.BindingEnricher{
		gremlins.NewEnricher(),
		target.NewEnricher(),
		openkneeboard.NewEnricher(),
		srs.NewEnricher(),
	}

	// Process all configured simulators
	workflow.ExportAllSimulatorsBatchWithInterfaces(parsers, enrichers, filter, noSVG)
}

// runInteractiveMode executes the interactive configuration workflow
func runInteractiveMode(config *common.Config) error {
	// Select simulator
	simType := common.SelectSimulation()
	configPath := common.GetConfigPath(config, simType, false)

	// Save the config path for batch mode reuse
	saveSimulatorPath(config, simType, configPath)

	// Configure optional tools
	configureOptionalTools(config)

	// Parse simulator files
	profiles, err := parseSimulator(config, simType, configPath)
	if err != nil {
		return err
	}

	// Display results
	common.DisplayProfiles(profiles)

	// Interactive configuration workflow
	workflow.ConfigureWorkflowInteractive(profiles, simType)
	return nil
}

// saveSimulatorPath saves the simulator path to the configuration
func saveSimulatorPath(config *common.Config, simType common.SimulationType, configPath string) {
	switch simType {
	case common.IL2Sturmovik, common.IL2Korea:
		simConfig := config.GetSimulatorConfig(simType)
		simConfig.IL2InputPath = configPath
		if err := common.SaveConfig(config); err != nil {
			fmt.Printf("⚠ Unable to save %s path: %v\n", simType, err)
		}
	case common.DCSWorld:
		dcsConfig := config.GetSimulatorConfig(common.DCSWorld)
		dcsConfig.DCSPath = configPath
		if err := common.SaveConfig(config); err != nil {
			fmt.Printf("⚠ Unable to save DCS path: %v\n", err)
		}
	}
}

// configureOptionalTools prompts the user to configure optional tools
func configureOptionalTools(config *common.Config) {
	// Configure draw.io for PNG export (optional)
	if _, found := common.VerifyDrawIOPath(config); !found {
		if !common.ConfigureDrawIOPath(config) {
			fmt.Println("\n⚠ PNG export will be disabled (draw.io not configured)")
		}
	}

	// Configure optional components using the Configurable interface
	configurables := []common.Configurable{
		srs.NewConfigurator(),
		openkneeboard.NewConfigurator(),
	}

	for _, cfg := range configurables {
		if err := cfg.Configure(config, false); err != nil {
			fmt.Printf("\n⚠ %s configuration error: %v\n", cfg.GetName(), err)
		}
	}
}

// parseSimulator parses the simulator configuration files using the appropriate parser
func parseSimulator(config *common.Config, simType common.SimulationType, configPath string) (*common.ProfileCollection, error) {
	// Get the appropriate parser for the simulator type
	var parser common.SimulatorParser

	switch simType {
	case common.DCSWorld:
		parser = dcs.NewParser()
	case common.IL2Sturmovik:
		parser = il2.NewParser()
	case common.IL2Korea:
		parser = il2korea.NewParser(config)
	default:
		return nil, fmt.Errorf("simulation type not supported")
	}

	// Parse using the interface
	profiles, err := parser.Parse(configPath)
	if err != nil {
		return nil, fmt.Errorf("error during parsing: %v", err)
	}

	return profiles, nil
}
