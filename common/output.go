package common

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Progress output for the export pipeline (workflow, csv, svg) goes through
// this writer instead of straight to stdout, so a GUI can capture it and stream
// it to a log panel. The interactive CLI prompts deliberately keep using fmt
// directly: they are tied to the terminal anyway.
// A nil writer means "whatever os.Stdout is right now". Resolving it lazily
// matters on Windows: a GUI-subsystem binary reassigns os.Stdout when it
// reattaches to the parent console, and a value captured at init would keep
// pointing at the dead handle.
var (
	outMu sync.RWMutex
	out   io.Writer
)

// SetOutput redirects the export progress output. Passing nil restores stdout.
func SetOutput(w io.Writer) {
	outMu.Lock()
	defer outMu.Unlock()
	out = w
}

// Output returns the current progress writer.
func Output() io.Writer {
	outMu.RLock()
	defer outMu.RUnlock()
	if out == nil {
		return os.Stdout
	}
	return out
}

// Printf writes formatted progress output.
func Printf(format string, a ...any) {
	fmt.Fprintf(Output(), format, a...)
}

// Println writes progress output followed by a newline.
func Println(a ...any) {
	fmt.Fprintln(Output(), a...)
}
