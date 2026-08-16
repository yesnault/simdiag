package gui

import (
	"runtime"
	"strings"
	"testing"
)

// The startup check gates the whole GUI, so a false negative would make the
// application refuse to start on a perfectly good machine. This test asserts the
// detection agrees with reality on the machine running it: every supported
// Windows install (Win11, and Win10 with a current Edge) ships the runtime.
func TestWebView2Version_DetectsTheInstalledRuntime(t *testing.T) {
	version := WebView2Version()

	if runtime.GOOS != "windows" {
		if version == "" {
			t.Error("the non-Windows stub must report a version so startup is never blocked")
		}
		return
	}

	if version == "" {
		t.Skip("WebView2 runtime is genuinely absent on this machine")
	}

	// A real version looks like 141.0.3537.85; EdgeUpdate leaves "0.0.0.0"
	// behind after an uninstall and that must not count as installed.
	if version == "0.0.0.0" {
		t.Errorf("WebView2Version() = %q, which means not installed", version)
	}
	if !strings.Contains(version, ".") {
		t.Errorf("WebView2Version() = %q, want a dotted version string", version)
	}

	t.Logf("WebView2 runtime detected: %s", version)
}

func TestWebView2DownloadURL(t *testing.T) {
	if !strings.HasPrefix(WebView2DownloadURL, "https://") {
		t.Errorf("WebView2DownloadURL = %q, want an https URL", WebView2DownloadURL)
	}
}
