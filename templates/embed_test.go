package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNames_AreRootSVGFilesOnly(t *testing.T) {
	names := Names()

	if len(names) == 0 {
		t.Fatal("no base template is embedded")
	}

	for _, name := range names {
		if !strings.EqualFold(filepath.Ext(name), ".svg") {
			t.Errorf("%q is embedded but is not an .svg", name)
		}
		// The bespoke button boxes live in a subdirectory and must stay out of
		// the binary: they are one pilot's hardware.
		if strings.ContainsAny(name, `/\`) {
			t.Errorf("%q comes from a subdirectory, which must not be embedded", name)
		}
	}
}

func TestRead_ReturnsTheTemplate(t *testing.T) {
	name := Names()[0]

	data, err := Read(name)
	if err != nil {
		t.Fatalf("Read(%q): %v", name, err)
	}
	if !strings.Contains(string(data[:min(len(data), 4096)]), "<svg") {
		t.Errorf("%q does not look like an SVG", name)
	}
}

func TestInstall_WritesEveryTemplateIntoAnEmptyDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "templates")

	written, err := Install(dir)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if len(written) != len(Names()) {
		t.Errorf("wrote %d template(s), want all %d", len(written), len(Names()))
	}
	for _, name := range Names() {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%q was not written: %v", name, err)
		}
	}
	if missing := Missing(dir); len(missing) != 0 {
		t.Errorf("still missing %v after installing", missing)
	}
}

func TestInstall_IsANoOpTheSecondTime(t *testing.T) {
	dir := t.TempDir()

	if _, err := Install(dir); err != nil {
		t.Fatalf("first Install: %v", err)
	}

	written, err := Install(dir)
	if err != nil {
		t.Fatalf("second Install: %v", err)
	}
	if len(written) != 0 {
		t.Errorf("second run wrote %v, want nothing", written)
	}
}

// A template someone has edited is their work. Installing must never take it
// back, which is the whole reason Install only writes what is missing.
func TestInstall_LeavesAnEditedTemplateAlone(t *testing.T) {
	dir := t.TempDir()
	edited := Names()[0]
	const mine = "<svg>mine</svg>"

	if err := os.WriteFile(filepath.Join(dir, edited), []byte(mine), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	written, err := Install(dir)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	for _, name := range written {
		if name == edited {
			t.Errorf("%q was overwritten", edited)
		}
	}

	after, err := os.ReadFile(filepath.Join(dir, edited))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(after) != mine {
		t.Errorf("%q = %d bytes, want the edited %d bytes kept", edited, len(after), len(mine))
	}
}

func TestMissing_ReportsEverythingForADirectoryThatIsNotThere(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "nope")

	if got := Missing(absent); len(got) != len(Names()) {
		t.Errorf("Missing(absent) returned %d, want all %d", len(got), len(Names()))
	}
}

func TestInstall_RefusesWithoutADirectory(t *testing.T) {
	if _, err := Install(""); err == nil {
		t.Error("Install(\"\") should fail: there is nowhere to write")
	}
}
