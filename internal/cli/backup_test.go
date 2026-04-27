package cli

import (
	"os"
	"testing"
)

func TestIsInteractiveInputFalseWhenStdinIsPipe(t *testing.T) {
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
		_ = w.Close()
	}()
	if isInteractiveInput() {
		t.Fatal("expected pipe stdin to be non-interactive")
	}
}
