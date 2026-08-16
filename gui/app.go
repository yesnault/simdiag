// Package gui hosts the graphical front end. Every Wails import in the project
// lives in this package: Wails v3 is still a beta, and confining it here keeps
// the blast radius of an API change to one directory.
package gui

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// wailsApp is the running application, kept so the About tab can restart it
// after an update. It is the only state this package holds outside State.
var wailsApp *application.App

// Run starts the graphical application and blocks until the window is closed.
//
// With no configuration named on the command line, it reopens the file the user
// last worked on, falling back to the conventional location for a first run.
func Run(version string) error {
	if recent := mostRecentExisting(); recent != "" {
		return RunWithConfig(version, recent)
	}
	return RunWithConfig(version, DefaultConfigPath())
}

// RunWithConfig starts the graphical application on an explicit configuration
// file. A relative path is resolved against the working directory, so
// simdiag.exe -c users/yesno/mapping_config_yesno.yaml opens that profile.
func RunWithConfig(version, configPath string) error {
	// Without the WebView2 runtime there is no window to report anything in, so
	// say so in a native dialog before Wails fails further down.
	if WebView2Version() == "" {
		message := "SimDiag needs the Microsoft Edge WebView2 Runtime, which is not installed.\n\n" +
			"Install it from:\n" + WebView2DownloadURL + "\n\n" +
			"The command line still works without it, for example:\n    simdiag.exe -b"
		ShowErrorDialog("SimDiag: WebView2 Runtime missing", message)
		return fmt.Errorf("WebView2 runtime not installed: see %s", WebView2DownloadURL)
	}

	// The GUI needs the same cross-package wiring as the CLI, or the enrichers
	// nil-panic partway through an export.

	if abs, err := filepath.Abs(configPath); err == nil {
		configPath = abs
	}

	// Same entry point the runtime switch uses (State.SwitchTo), so startup and
	// a later change of file cannot end up disagreeing about the working
	// directory or common.ConfigFileName.
	if err := enterConfigDirectory(configPath); err != nil {
		return err
	}

	state, err := NewState(configPath, version)
	if err != nil {
		return fmt.Errorf("unable to load configuration from %s: %w", configPath, err)
	}
	rememberRecent(configPath)

	wailsApp = application.New(application.Options{
		Name:        "SimDiag",
		Description: "Simulator controller diagram generator",
		Assets: application.AssetOptions{
			Handler:        NewHandler(state),
			DisableLogging: true,
		},
	})

	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "SimDiag " + version,
		Width:     1400,
		Height:    900,
		MinWidth:  1000,
		MinHeight: 640,
		URL:       "/",
		// The native window paints before the first frame; without this it
		// flashes white on startup. Tracks --bg in gui/frontend/app.css.
		BackgroundColour: application.NewRGB(0x1c, 0x1c, 0x1c),
	})

	// Wails does not carry the executable's icon onto its own windows. Hook the
	// show event rather than window creation: the native handle only exists once
	// the window is actually displayed.
	window.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		applyWindowIcon()
	})

	return wailsApp.Run()
}

// restartApplication launches exePath and closes this instance.
//
// exePath is the path the update installed at, captured before the swap: once
// the running image has been renamed aside, os.Executable can answer with the
// backup's name instead. The working directory is set to the executable's own,
// rather than inherited. The process has chdir'd into the configuration's
// directory, and the new instance reopens the recent configuration by itself.
func restartApplication(exePath string) error {
	command := exec.Command(exePath)
	command.Dir = filepath.Dir(exePath)

	if err := command.Start(); err != nil {
		return fmt.Errorf("unable to start %s: %w", exePath, err)
	}

	// Quitting after the new process is up: if the launch fails, the user is
	// left with a working window rather than with nothing.
	if wailsApp != nil {
		go wailsApp.Quit()
	}

	return nil
}
