package gui

import (
	"bytes"
	"context"
	"sync"

	"simdiag/workflow"
)

// lineWriter turns the export's byte stream into whole lines, so the frontend
// receives one log entry per line instead of arbitrary chunks. It is the bridge
// between common.SetOutput (an io.Writer) and the run's log buffer.
type lineWriter struct {
	emit func(string)

	mu      sync.Mutex
	partial bytes.Buffer
}

func newLineWriter(emit func(string)) *lineWriter {
	return &lineWriter{emit: emit}
}

// Write buffers until a newline, emitting each complete line.
func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.partial.Write(p)

	for {
		data := w.partial.Bytes()
		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			break
		}

		line := string(bytes.TrimRight(data[:i], "\r"))
		w.partial.Next(i + 1)
		w.emit(line)
	}

	return len(p), nil
}

// Flush emits whatever was written without a trailing newline.
func (w *lineWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.partial.Len() > 0 {
		w.emit(w.partial.String())
		w.partial.Reset()
	}
}

// exportRun is the state of the one export the application allows at a time.
//
// The frontend starts a run and then polls for new lines rather than reading a
// streaming response: Wails' asset server buffers a handler's response until it
// returns, so a long-lived streaming request delivers nothing until the export
// is already over.
type exportRun struct {
	mu        sync.Mutex
	running   bool
	lines     []string
	summary   *workflow.ExportSummary
	failure   string
	cancelled bool
	cancel    context.CancelFunc
}

// begin marks a run as started, or reports false if one is already in flight.
func (r *exportRun) begin(cancel context.CancelFunc) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return false
	}

	r.running = true
	r.lines = nil
	r.summary = nil
	r.failure = ""
	r.cancelled = false
	r.cancel = cancel

	return true
}

// isRunning reports whether an export is in flight, without claiming the slot.
func (r *exportRun) isRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// appendLine records one line of progress output.
func (r *exportRun) appendLine(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines = append(r.lines, line)
}

// finish records the outcome and releases the run.
func (r *exportRun) finish(summary *workflow.ExportSummary, failure string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.running = false
	r.summary = summary
	r.failure = failure
	r.cancel = nil
}

// requestCancel stops a running export. The pipeline honours the context at
// every simulator and every diagram.
func (r *exportRun) requestCancel() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running || r.cancel == nil {
		return false
	}

	r.cancelled = true
	r.cancel()

	return true
}

// exportState is the polling response: whatever is new since the caller's
// position in the log, plus the run's status.
type exportState struct {
	Running   bool                    `json:"running"`
	Cancelled bool                    `json:"cancelled"`
	Lines     []string                `json:"lines"`
	NextIndex int                     `json:"nextIndex"`
	Summary   *workflow.ExportSummary `json:"summary,omitempty"`
	Error     string                  `json:"error,omitempty"`
}

// stateSince returns the run's status and the log lines after index from.
func (r *exportRun) stateSince(from int) exportState {
	r.mu.Lock()
	defer r.mu.Unlock()

	if from < 0 || from > len(r.lines) {
		from = 0
	}

	// Always a list, never null: a poll that finds nothing new is the common
	// case, and JSON null is not iterable on the other side.
	lines := make([]string, 0, len(r.lines)-from)
	lines = append(lines, r.lines[from:]...)

	return exportState{
		Running:   r.running,
		Cancelled: r.cancelled,
		Lines:     lines,
		NextIndex: len(r.lines),
		Summary:   r.summary,
		Error:     r.failure,
	}
}
