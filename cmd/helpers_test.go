package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestPageFlags_Values(t *testing.T) {
	cases := []struct {
		name      string
		page      int
		limit     int
		wantPage  string
		wantLimit string
	}{
		{"both set", 2, 25, "2", "25"},
		{"only page", 3, 0, "3", ""},
		{"only limit", 0, 50, "", "50"},
		{"neither set", 0, 0, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pf := pageFlags{page: tc.page, limit: tc.limit}
			q := pf.values()
			if got := q.Get("page"); got != tc.wantPage {
				t.Errorf("page = %q, want %q", got, tc.wantPage)
			}
			if got := q.Get("limit"); got != tc.wantLimit {
				t.Errorf("limit = %q, want %q", got, tc.wantLimit)
			}
		})
	}
}

func TestPageFlags_BindRegistersFlagsWithDefaults(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var pf pageFlags
	pf.bind(cmd)

	pageFlag := cmd.Flag("page")
	limitFlag := cmd.Flag("limit")
	if pageFlag == nil || limitFlag == nil {
		t.Fatal("--page or --limit not registered")
	}
	if pageFlag.DefValue != "1" {
		t.Errorf("page default = %q, want 1", pageFlag.DefValue)
	}
	if limitFlag.DefValue != "50" {
		t.Errorf("limit default = %q, want 50", limitFlag.DefValue)
	}
}

func TestNewOpCtx_AppliesTimeout(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())

	ctx, cancel := newOpCtx(cmd, 50*time.Millisecond)
	defer cancel()
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		t.Error("expected context to have a deadline")
	}
	select {
	case <-ctx.Done():
		// expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("context didn't fire within 500ms")
	}
}

func TestNewOpCtx_ZeroTimeoutGivesCancelOnlyContext(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	ctx, cancel := newOpCtx(cmd, 0)
	defer cancel()
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		t.Error("expected no deadline when timeout is zero")
	}
	if ctx.Err() != nil {
		t.Errorf("context already done: %v", ctx.Err())
	}
}

func TestNewOpCtx_NilCommandContextFallsBackToBackground(t *testing.T) {
	cmd := &cobra.Command{Use: "test"} // no SetContext call

	ctx, cancel := newOpCtx(cmd, 10*time.Millisecond)
	defer cancel()
	if ctx == nil {
		t.Fatal("ctx is nil")
	}
	// Should still respect the timeout we passed.
	if _, ok := ctx.Deadline(); !ok {
		t.Error("expected deadline even when parent context is nil")
	}
}
