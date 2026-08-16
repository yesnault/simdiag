//go:build windows

package gui

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// WebView2DownloadURL is where a user without the runtime gets it.
const WebView2DownloadURL = "https://developer.microsoft.com/microsoft-edge/webview2/"

// webView2ClientGUID identifies the Evergreen WebView2 Runtime in EdgeUpdate.
const webView2ClientGUID = `{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`

// WebView2Version returns the installed WebView2 Runtime version, or "" when the
// runtime is absent.
//
// Wails detects this too, but only from an internal package we cannot import,
// and only once the window is already being created, by which point a missing
// runtime is an obscure failure rather than an explanation. Checking up front
// lets the application say what is wrong and where to fix it.
//
// The registry locations are the ones Microsoft documents for runtime detection.
func WebView2Version() string {
	locations := []struct {
		key  registry.Key
		path string
	}{
		// 64-bit Windows records a machine-wide install under WOW6432Node.
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\` + webView2ClientGUID},
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\EdgeUpdate\Clients\` + webView2ClientGUID},
		// Per-user installs.
		{registry.CURRENT_USER, `Software\Microsoft\EdgeUpdate\Clients\` + webView2ClientGUID},
	}

	for _, location := range locations {
		key, err := registry.OpenKey(location.key, location.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		version, _, err := key.GetStringValue("pv")
		key.Close()

		// EdgeUpdate leaves a "0.0.0.0" entry behind after an uninstall.
		if err == nil && version != "" && version != "0.0.0.0" {
			return version
		}
	}

	return ""
}

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW = user32.NewProc("MessageBoxW")
)

// MB_OK | MB_ICONERROR | MB_SETFOREGROUND
const messageBoxErrorFlags = 0x00000000 | 0x00000010 | 0x00010000

// ShowErrorDialog displays a native message box. It is the only way to reach a
// user whose window never opened: the process has no console and no webview.
func ShowErrorDialog(title, message string) {
	titleUTF16, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	messageUTF16, err := syscall.UTF16PtrFromString(message)
	if err != nil {
		return
	}

	_, _, _ = procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(messageUTF16)),
		uintptr(unsafe.Pointer(titleUTF16)),
		messageBoxErrorFlags,
	)
}
