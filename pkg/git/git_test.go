package git

import (
	"testing"
)

func TestCaptureContext(t *testing.T) {
	ctx, err := CaptureContext(".")
	if err != nil {
		t.Fatalf("CaptureContext() error = %v", err)
	}

	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	// We are running in a git repo
	if ctx.Repository == "" {
		t.Log("Note: git repo root not detected (could be outside git worktree)")
	}
}
