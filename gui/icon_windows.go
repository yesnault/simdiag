//go:build windows

package gui

import (
	"syscall"
	"unsafe"
)

// The icon compiled into the executable, named by winres/winres.json.
const iconResourceName = "APP"

const (
	imageIcon      = 1
	wmSetIcon      = 0x0080
	iconSmall      = 0
	iconBig        = 1
	smCXIcon       = 11
	smCXSmallIcon  = 49
	lrDefaultColor = 0x0000
	lrShared       = 0x8000
)

var (
	kernel32DLL          = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleHandleW = kernel32DLL.NewProc("GetModuleHandleW")
	procGetCurrentPID    = kernel32DLL.NewProc("GetCurrentProcessId")

	user32DLL              = syscall.NewLazyDLL("user32.dll")
	procLoadImageW         = user32DLL.NewProc("LoadImageW")
	procSendMessageW       = user32DLL.NewProc("SendMessageW")
	procEnumWindows        = user32DLL.NewProc("EnumWindows")
	procGetWindowThreadPID = user32DLL.NewProc("GetWindowThreadProcessId")
	procGetSystemMetrics   = user32DLL.NewProc("GetSystemMetrics")
)

// applyWindowIcon puts the executable's icon on every top-level window this
// process owns.
//
// Wails v3 beta registers its window class with IDI_APPLICATION, the generic
// system icon, and its Windows setIcon is an empty stub, so Options.Icon does
// nothing. Without this, the title bar and taskbar show a placeholder even
// though the icon is compiled into the binary and Explorer displays it.
//
// Best effort throughout: a missing icon is a cosmetic problem, never a reason
// to stop the application.
func applyWindowIcon() {
	instance, _, _ := procGetModuleHandleW.Call(0)
	if instance == 0 {
		return
	}

	big := loadIcon(instance, smCXIcon)
	small := loadIcon(instance, smCXSmallIcon)
	if big == 0 && small == 0 {
		return
	}

	currentPID, _, _ := procGetCurrentPID.Call()

	callback := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var pid uint32
		_, _, _ = procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

		if uintptr(pid) == currentPID {
			if small != 0 {
				_, _, _ = procSendMessageW.Call(hwnd, wmSetIcon, iconSmall, small)
			}
			if big != 0 {
				_, _, _ = procSendMessageW.Call(hwnd, wmSetIcon, iconBig, big)
			}
		}

		return 1 // keep enumerating
	})

	_, _, _ = procEnumWindows.Call(callback, 0)
}

// loadIcon loads the named icon resource at the system size given by a
// GetSystemMetrics index.
func loadIcon(instance uintptr, metric int) uintptr {
	size, _, _ := procGetSystemMetrics.Call(uintptr(metric))

	name, err := syscall.UTF16PtrFromString(iconResourceName)
	if err != nil {
		return 0
	}

	icon, _, _ := procLoadImageW.Call(
		instance,
		uintptr(unsafe.Pointer(name)),
		imageIcon,
		size,
		size,
		lrDefaultColor|lrShared,
	)

	return icon
}
