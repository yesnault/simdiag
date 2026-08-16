//go:build !windows

package app

// AttachParentConsole is a no-op outside Windows: every other platform hands a
// process its standard handles regardless of how it was launched.
func AttachParentConsole() {}
