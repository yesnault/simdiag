package gui

import (
	"errors"
	"testing"
)

// The Windows common file dialog reports a cancellation as an error, and this
// is the only thing standing between the user pressing Cancel and a red failure
// message. The sentinel is unexported by Wails, so the message is pinned here.
func TestDialogCancelled(t *testing.T) {
	if !dialogCancelled(errors.New("cancelled by user")) {
		t.Error("the Windows picker's cancellation was not recognised")
	}
	if dialogCancelled(errors.New("access denied")) {
		t.Error("a real failure was mistaken for a cancellation")
	}
	if dialogCancelled(nil) {
		t.Error("no error is not a cancellation")
	}
}
