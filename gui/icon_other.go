//go:build !windows

package gui

// applyWindowIcon is Windows-only: elsewhere the icon comes from the desktop
// entry or the bundle rather than from a resource inside the executable.
func applyWindowIcon() {}
