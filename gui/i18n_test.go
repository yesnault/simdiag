package gui

import (
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The catalogue is JavaScript and there is no JS tooling in this project, so
// these tests read the frontend files out of the embedded filesystem and check
// them from Go. A missing translation is otherwise invisible until the key shows
// up verbatim in the interface.

var (
	// "key": { en: "...", fr: "..." } is the entry, however it is wrapped across
	// lines. The body may itself contain braces: the texts carry {placeholder}
	// substitutions, so one level of nesting has to be allowed for.
	catalogueEntry = regexp.MustCompile(`"([\w.]+)":\s*\{((?:[^{}]|\{[^{}]*\})*)\}`)
	englishText    = regexp.MustCompile(`(?s)\ben:\s*"((?:[^"\\]|\\.)*)"`)
	frenchText     = regexp.MustCompile(`(?s)\bfr:\s*"((?:[^"\\]|\\.)*)"`)

	// t("key", ...) and setStatus("key", ...) are the two ways the page names a
	// message; nothing else takes a catalogue key.
	usedInJS   = regexp.MustCompile(`\b(?:t|setStatus)\(\s*"([\w.]+)"`)
	usedInHTML = regexp.MustCompile(`data-i18n(?:-title|-placeholder)?="([\w.]+)"`)

	// Placeholders both sides must agree on, e.g. {count}.
	placeholder = regexp.MustCompile(`\{(\w+)\}`)
)

type catalogue map[string]struct{ en, fr string }

func readFrontend(t *testing.T, name string) string {
	t.Helper()

	data, err := frontendFS.ReadFile("frontend/" + name)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(data)
}

// keysUsedByTheInterface collects every catalogue key the page names, from all
// of the frontend rather than from one file.
//
// The scan is by glob rather than by a list of names: the interface is split
// across one module per tab, and a list would silently stop covering the next
// file added. i18n.js is scanned along with the rest and contributes nothing —
// its entries are written "key": { … }, never t("key").
func keysUsedByTheInterface(t *testing.T) map[string]string {
	t.Helper()

	scripts, err := fs.Glob(frontendFS, "frontend/*.js")
	if err != nil {
		t.Fatalf("listing the frontend scripts: %v", err)
	}
	if len(scripts) == 0 {
		t.Fatal("no frontend script found, the scan is not working")
	}

	used := map[string]string{} // key -> the file naming it
	for _, script := range scripts {
		name := path.Base(script)
		for _, match := range usedInJS.FindAllStringSubmatch(readFrontend(t, name), -1) {
			used[match[1]] = name
		}
	}
	for _, match := range usedInHTML.FindAllStringSubmatch(readFrontend(t, "index.html"), -1) {
		used[match[1]] = "index.html"
	}

	return used
}

func loadCatalogue(t *testing.T) catalogue {
	t.Helper()

	source := readFrontend(t, "i18n.js")
	// Everything after the catalogue is code, and it must not be scanned for
	// entries: the helpers below it contain braces of their own.
	if end := strings.Index(source, "\n};"); end > 0 {
		source = source[:end]
	}

	entries := make(catalogue)
	for _, match := range catalogueEntry.FindAllStringSubmatch(source, -1) {
		key, body := match[1], match[2]

		var entry struct{ en, fr string }
		if en := englishText.FindStringSubmatch(body); en != nil {
			entry.en = en[1]
		}
		if fr := frenchText.FindStringSubmatch(body); fr != nil {
			entry.fr = fr[1]
		}
		entries[key] = entry
	}

	if len(entries) < 50 {
		t.Fatalf("only %d catalogue entries parsed, the catalogue is far larger", len(entries))
	}
	return entries
}

func TestCatalogue_EveryEntryHasBothLanguages(t *testing.T) {
	for key, entry := range loadCatalogue(t) {
		if strings.TrimSpace(entry.en) == "" {
			t.Errorf("%s has no English text", key)
		}
		if strings.TrimSpace(entry.fr) == "" {
			t.Errorf("%s has no French text", key)
		}
	}
}

// A placeholder that exists on one side only renders as a literal "{count}" in
// that language, which is the kind of thing nobody notices until a user does.
func TestCatalogue_BothLanguagesUseTheSamePlaceholders(t *testing.T) {
	for key, entry := range loadCatalogue(t) {
		en := placeholdersOf(entry.en)
		fr := placeholdersOf(entry.fr)

		if strings.Join(en, ",") != strings.Join(fr, ",") {
			t.Errorf("%s: English uses %v, French uses %v", key, en, fr)
		}
	}
}

func placeholdersOf(text string) []string {
	var names []string
	for _, match := range placeholder.FindAllStringSubmatch(text, -1) {
		names = append(names, match[1])
	}
	sort.Strings(names)
	return names
}

func TestCatalogue_CoversEveryKeyTheInterfaceUses(t *testing.T) {
	entries := loadCatalogue(t)

	used := keysUsedByTheInterface(t)

	if len(used) < 50 {
		t.Fatalf("only %d keys found in the frontend, the scan is not working", len(used))
	}

	for key, where := range used {
		if _, ok := entries[key]; !ok {
			t.Errorf("%s uses %q, which the catalogue does not have", where, key)
		}
	}
}

// Codes Go emits are rendered by the same catalogue, and nothing else would
// catch a typo in one of them until it reached the screen.
func TestCatalogue_CoversEveryCodeGoEmits(t *testing.T) {
	entries := loadCatalogue(t)

	for _, code := range goEmittedCodes() {
		if _, ok := entries[code]; !ok {
			t.Errorf("Go emits %q, which the catalogue does not have", code)
		}
	}
}

func TestCatalogue_HasNoUnusedEntries(t *testing.T) {
	entries := loadCatalogue(t)

	used := map[string]bool{}
	for key := range keysUsedByTheInterface(t) {
		used[key] = true
	}
	for _, code := range goEmittedCodes() {
		used[code] = true
	}

	for key := range entries {
		if !used[key] {
			t.Errorf("the catalogue has %q, which nothing uses", key)
		}
	}
}

// goEmittedCodes lists the codes reached through message values rather than
// through a t("...") call, so the unused-entry check does not flag them.
func goEmittedCodes() []string {
	return []string{
		msgNotConfigured, msgTemplatesRequired, msgTemplatesDirNotFound,
		msgTemplatesUnreadable, msgTemplatesNone, msgTemplatesFound,
		msgOutputRequired, msgOutputWillBeCreated, msgDrawIONotFound,
		msgOpenKneeboardNotFound, msgSRSNotFound, msgGremlinsNotFound,
		msgTargetNotFound, msgSimulatorNotConfigured, msgSimulatorDirNotFound,
		msgParserFailed, msgTemplatesUnconfigured, msgTemplatesReadFailed,
		msgOutputUnconfigured, msgOutputReadFailed, msgNoDiagramYet, msgOutputGroup,
		msgExportTargetEverything, msgExportNotConfigured,
		msgExportRunning, msgNoConfigFile, msgNoCSVToRegen, msgFolderMissing,
		msgTargetDetectFailed,
		msgBatchScriptFailed, msgUpdateCheckFailed, msgUpdateRunning,
		msgAlreadyCurrent, msgNothingInstalled, msgRestartFailed,
	}
}

// The native OS dialogs are the one thing the frontend cannot draw, so their
// captions are translated in Go instead, and must be complete too.
func TestDialogTexts_AreComplete(t *testing.T) {
	for key, texts := range dialogTexts {
		if strings.TrimSpace(texts[languageEnglish]) == "" {
			t.Errorf("%s has no English caption", key)
		}
		if strings.TrimSpace(texts[languageFrench]) == "" {
			t.Errorf("%s has no French caption", key)
		}
	}

	if got := dialogText("dialog.open.title"); got == "" {
		t.Error("dialogText returned nothing for a known key")
	}
	if got := dialogText("nope"); got != "nope" {
		t.Errorf("dialogText(unknown) = %q, want the key back", got)
	}
}
