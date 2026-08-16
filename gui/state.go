package gui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"

	"simdiag/common"
)

// State is the single mutable thing the GUI owns: the loaded configuration and
// where it came from. Every HTTP handler reads it through the mutex, since the
// asset server serves requests on its own goroutines.
type State struct {
	mu         sync.RWMutex
	configPath string
	config     *common.Config
	version    string
}

// NewState loads the configuration for the given path. A missing file is not an
// error: it yields an empty config, which is how a first run starts.
func NewState(configPath, version string) (*State, error) {
	config, err := common.LoadConfigFrom(configPath)
	if err != nil {
		return nil, err
	}

	return &State{
		configPath: configPath,
		config:     config,
		version:    version,
	}, nil
}

// errNoConfigFile is the one failure of SwitchTo the user is meant to act on, so
// the route reports it with a translated message rather than with this text.
var errNoConfigFile = errors.New("no configuration file")

// SwitchTo replaces the loaded configuration with another file.
//
// This is the only place the application changes configuration, startup
// included, because a switch is more than a new struct: the process has
// to move to the new file's directory, since configurations name their
// templates and output relatively to themselves (templates_directory:
// ./templates), and the CLI-shared global common.ConfigFileName has to follow.
//
// The file is read before anything is mutated, so a bad path leaves the running
// configuration untouched.
func (s *State) SwitchTo(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid configuration path %s: %w", path, err)
	}

	// Unlike NewState, where a missing file is a legitimate first run, being
	// asked to switch to a file that is not there is an error worth reporting.
	if info, err := os.Stat(abs); err != nil || info.IsDir() {
		return fmt.Errorf("%w: %s", errNoConfigFile, abs)
	}

	config, err := common.LoadConfigFrom(abs)
	if err != nil {
		return fmt.Errorf("unable to load configuration from %s: %w", abs, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := enterConfigDirectory(abs); err != nil {
		return err
	}

	s.configPath = abs
	s.config = config

	return nil
}

// enterConfigDirectory moves the process to a configuration's directory and
// points the CLI-shared global at the file.
//
// Configuration files use paths relative to themselves, which the CLI resolves
// because it is run from that folder. A windowed application inherits an
// arbitrary working directory, so moving there makes every relative path mean
// the same thing in both front ends.
func enterConfigDirectory(configPath string) error {
	dir := filepath.Dir(configPath)

	// A first run points at a configuration that does not exist yet, and
	// %APPDATA%\simdiag is not created until something is saved there.
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("unable to create the configuration directory %s: %w", dir, err)
	}
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("unable to enter the configuration directory %s: %w", dir, err)
	}

	common.SetConfigFileName(configPath)

	return nil
}

// Config returns the loaded configuration.
func (s *State) Config() *common.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// ConfigPath returns the path the configuration is read from and written to.
func (s *State) ConfigPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configPath
}

// Version returns the SimDiag version string.
func (s *State) Version() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// ConfigSnapshot returns a deep copy of the configuration.
//
// An export runs for seconds while the user can still hit Save in the
// Configuration tab, which writes into the very struct the export is reading,
// including maps, so the race is not theoretical. Round-tripping through YAML is
// the copy the config type already supports.
func (s *State) ConfigSnapshot() (*common.Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := yaml.Marshal(s.config)
	if err != nil {
		return nil, fmt.Errorf("unable to copy the configuration: %w", err)
	}

	var copied common.Config
	if err := yaml.Unmarshal(data, &copied); err != nil {
		return nil, fmt.Errorf("unable to copy the configuration: %w", err)
	}
	if copied.Simulators == nil {
		copied.Simulators = make(map[string]*common.SimulatorConfig)
	}

	return &copied, nil
}

// Save writes the in-memory configuration to disk.
func (s *State) Save() error {
	s.mu.RLock()
	config, path := s.config, s.configPath
	s.mu.RUnlock()

	return common.SaveConfigTo(config, path)
}
