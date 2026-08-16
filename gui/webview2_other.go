//go:build !windows

package gui

// WebView2DownloadURL is unused outside Windows, where the webview comes from
// WebKit rather than a separately installed runtime.
const WebView2DownloadURL = "https://developer.microsoft.com/microsoft-edge/webview2/"

// WebView2Version reports a non-empty value so the startup check never blocks a
// non-Windows build.
func WebView2Version() string { return "n/a" }

// ShowErrorDialog has no native equivalent worth wiring up here; the error is
// returned to the caller and printed instead.
func ShowErrorDialog(_, _ string) {}
