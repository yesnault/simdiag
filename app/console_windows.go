//go:build windows

package app

import (
	"os"
	"sync"
	"syscall"
)

// Win32 constants. All of these are DWORDs, so the negative sentinels are
// expressed as their 32-bit two's complement.
const (
	attachParentProcess = uintptr(0xFFFFFFFF) // ATTACH_PARENT_PROCESS, i.e. (DWORD)-1
	stdInputHandle      = uintptr(0xFFFFFFF6) // STD_INPUT_HANDLE,  (DWORD)-10
	stdOutputHandle     = uintptr(0xFFFFFFF5) // STD_OUTPUT_HANDLE, (DWORD)-11
	stdErrorHandle      = uintptr(0xFFFFFFF4) // STD_ERROR_HANDLE,  (DWORD)-12
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole    = kernel32.NewProc("AttachConsole")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	procGetStdHandle     = kernel32.NewProc("GetStdHandle")

	attachOnce sync.Once
)

// AttachParentConsole reconnects stdin/stdout/stderr to the console that
// launched this process.
//
// SimDiag is linked as a GUI-subsystem binary so that double-clicking it does
// not flash a console window. The side effect is that Windows hands such a
// process no standard handles: run simdiag.exe -b from a .bat and every Print
// goes nowhere, leaving the user staring at a blank window on pause.
// Reattaching to the parent console restores the CLI behaviour.
//
// Safe to call more than once. A no-op when the process already owns a console
// or has no parent console to attach to (launched from Explorer).
func AttachParentConsole() {
	attachOnce.Do(func() {
		// Already have a console? Leave the working handles alone.
		if hwnd, _, _ := procGetConsoleWindow.Call(); hwnd != 0 {
			return
		}

		if ret, _, _ := procAttachConsole.Call(attachParentProcess); ret == 0 {
			// No parent console: started from Explorer, or the parent has none.
			return
		}

		// Rebind only the streams Windows did not already give us. When the
		// caller redirected output (simdiag.exe -b > log.txt) or piped it, the
		// handle is valid and inherited. Overwriting it with CONOUT$ would send
		// the output to the console instead of the file.
		if !stdHandleValid(stdOutputHandle) {
			if f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
				os.Stdout = f
			}
		}
		if !stdHandleValid(stdErrorHandle) {
			if f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
				os.Stderr = f
			}
		}
		if !stdHandleValid(stdInputHandle) {
			if f, err := os.OpenFile("CONIN$", os.O_RDONLY, 0); err == nil {
				os.Stdin = f
			}
		}
	})
}

// stdHandleValid reports whether Windows already provided a usable handle for
// one of the standard streams.
func stdHandleValid(which uintptr) bool {
	h, _, _ := procGetStdHandle.Call(which)
	return h != 0 && h != ^uintptr(0) // NULL or INVALID_HANDLE_VALUE
}
