package gui

// The interface is translated in gui/frontend/i18n.js: Go names messages and the
// frontend renders them. This file is the one exception: the native Windows
// file dialogs are drawn by the OS, so their captions never pass through the
// page and have to be translated here.
//
// Keep it to dialog captions. Anything the frontend can draw belongs in the
// catalogue, where both languages sit side by side and the tests check them.
var dialogTexts = map[string]map[string]string{
	"dialog.open.title": {
		languageEnglish: "Open a SimDiag configuration",
		languageFrench:  "Ouvrir une configuration SimDiag",
	},
	"dialog.new.title": {
		languageEnglish: "New SimDiag configuration",
		languageFrench:  "Nouvelle configuration SimDiag",
	},
	"dialog.filter": {
		languageEnglish: "SimDiag configuration",
		languageFrench:  "Configuration SimDiag",
	},
}

// dialogText returns a native dialog caption in the user's language, falling
// back to English for a key or a language it does not know.
func dialogText(key string) string {
	texts, ok := dialogTexts[key]
	if !ok {
		return key
	}

	if text, ok := texts[loadSettings().Language]; ok {
		return text
	}
	return texts[languageEnglish]
}
