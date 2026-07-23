package prompt

import (
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestRenderPreviewSegments(t *testing.T) {
	enabled := config.SegmentConfig{Enabled: true}
	ctx := &fakeContext{
		Username:   "pilot",
		Cwd:        "~/repo",
		GitBranch:  "main",
		GitDirty:   true,
		ExitStatus: 9,
		Host:       "omega",
		Venv:       "venv",
		Node:       "v22.0.0",
		Go:         "go1.25.0",
		Battery:    "87%",
		Jobs:       2,
	}

	tests := []struct {
		name string
		fn   segmentFunc
		want string
	}{
		{name: "user", fn: renderUser, want: "pilot"},
		{name: "cwd", fn: renderCwd, want: "~/repo"},
		{name: "git", fn: renderGit, want: "main +"},
		{name: "status", fn: renderStatus, want: "✘ 9"},
		{name: "time", fn: renderTime, want: "14:52"},
		{name: "host", fn: renderHost, want: "omega"},
		{name: "venv", fn: renderVenv, want: "venv"},
		{name: "node", fn: renderNode, want: "v22.0.0"},
		{name: "go", fn: renderGo, want: "go1.25.0"},
		{name: "battery", fn: renderBattery, want: "87%"},
		{name: "jobs", fn: renderJobs, want: "2 jobs"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.fn(enabled, ctx); got != tc.want {
				t.Fatalf("%s segment = %q, want %q", tc.name, got, tc.want)
			}
			if got := tc.fn(config.SegmentConfig{Enabled: false}, ctx); got != "" {
				t.Fatalf("%s disabled segment = %q, want empty", tc.name, got)
			}
		})
	}
}

func TestRenderPreviewSegmentEmptyStates(t *testing.T) {
	enabled := config.SegmentConfig{Enabled: true}
	ctx := &fakeContext{}

	if got := renderGit(enabled, ctx); got != "" {
		t.Fatalf("renderGit(empty branch) = %q, want empty", got)
	}
	if got := renderStatus(enabled, ctx); got != "" {
		t.Fatalf("renderStatus(success hidden) = %q, want empty", got)
	}
	if got := renderStatus(config.SegmentConfig{Enabled: true, ShowSuccess: true}, ctx); got != "✓" {
		t.Fatalf("renderStatus(success shown) = %q, want check", got)
	}
	if got := renderJobs(enabled, ctx); got != "" {
		t.Fatalf("renderJobs(zero) = %q, want empty", got)
	}
}

func TestDefaultContextMatchesDefaultPreviewContext(t *testing.T) {
	ctx := defaultContext()
	preview := DefaultPreviewContext()

	if ctx.Username != preview.Username || ctx.Cwd != preview.Cwd || ctx.GitBranch != preview.GitBranch {
		t.Fatalf("defaultContext() = %+v, want values from %+v", ctx, preview)
	}
}
