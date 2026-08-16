package dcs

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// makeInputTree builds a Config/Input layout. A profile listed with joystick:true
// gets the joystick/ subfolder DCS creates once something is bound to a
// controller.
func makeInputTree(t *testing.T, profiles map[string]bool) string {
	t.Helper()

	base := t.TempDir()
	for name, joystick := range profiles {
		dir := filepath.Join(base, "Config", "Input", name)
		if joystick {
			dir = filepath.Join(dir, "joystick")
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("setup %s: %v", name, err)
		}
	}
	return base
}

func TestListModules(t *testing.T) {
	base := makeInputTree(t, map[string]bool{
		"M-2000C":       true,
		"FA-18C_hornet": true,
		"Bf-109K-4":     true,
		// Shared profiles: parsed, but not aircraft.
		"Default":     true,
		"UiLayer":     true,
		"CommandMenu": true,
		// The simplified scheme for an aircraft already listed.
		"M-2000C_easy": true,
		// Nothing bound to a controller: nothing to draw.
		"TF-51D": false,
	})

	modules, err := ListModules(base)
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}

	want := []string{"Bf-109K-4", "FA-18C_hornet", "M-2000C"}
	if !slices.Equal(modules, want) {
		t.Errorf("ListModules = %v, want %v", modules, want)
	}
}

// The cheap lister and the full parse must agree on what counts as a module, or
// the Generate dropdown offers a target the export then finds nothing for.
func TestListModules_AgreesWithParseDCS(t *testing.T) {
	base := makeInputTree(t, map[string]bool{
		"M-2000C":       true,
		"FA-18C_hornet": true,
		"P-47D-30_easy": true,
		"Default":       true,
		"UiLayer":       true,
		"TF-51D":        false,
	})

	listed, err := ListModules(base)
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}

	collection, err := parseDCS(base, "")
	if err != nil {
		t.Fatalf("parseDCS: %v", err)
	}

	parsed := []string{}
	for _, profile := range collection.Profiles {
		if profile.Module != "" {
			parsed = append(parsed, profile.Module)
		}
	}
	slices.Sort(parsed)

	if !slices.Equal(listed, parsed) {
		t.Errorf("ListModules = %v but parseDCS reports modules %v", listed, parsed)
	}
}

func TestListModules_MissingInputFolder(t *testing.T) {
	if _, err := ListModules(t.TempDir()); err == nil {
		t.Error("expected an error for a path with no Config/Input")
	}
}

func TestListModules_NoModules(t *testing.T) {
	base := makeInputTree(t, map[string]bool{"Default": true, "UiLayer": true})

	modules, err := ListModules(base)
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	if len(modules) != 0 {
		t.Errorf("ListModules = %v, want empty for a install with no aircraft", modules)
	}
}
