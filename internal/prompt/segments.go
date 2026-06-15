package prompt

import (
	"fmt"

	"github.com/snakepilot10/ozsh/internal/config"
)

type segmentFunc func(cfg config.SegmentConfig, ctx *fakeContext) string

var segmentRegistry = map[string]segmentFunc{
	"user":    renderUser,
	"cwd":     renderCwd,
	"git":     renderGit,
	"status":  renderStatus,
	"time":    renderTime,
	"host":    renderHost,
	"venv":    renderVenv,
	"node":    renderNode,
	"go":      renderGo,
	"battery": renderBattery,
	"jobs":    renderJobs,
}

type fakeContext struct {
	Username   string
	Cwd        string
	GitBranch  string
	GitDirty   bool
	ExitStatus int
	Host       string
	Venv       string
	Node       string
	Go         string
	Battery    string
	Jobs       int
}

type PreviewContext struct {
	Username   string
	Cwd        string
	GitBranch  string
	GitDirty   bool
	ExitStatus int
	Host       string
	Venv       string
	Node       string
	Go         string
	Battery    string
	Jobs       int
}

func defaultContext() *fakeContext {
	return contextFromPreview(DefaultPreviewContext())
}

func DefaultPreviewContext() PreviewContext {
	return PreviewContext{
		Username:   "snake",
		Cwd:        "~/dev/ozsh",
		GitBranch:  "main",
		GitDirty:   true,
		ExitStatus: 0,
		Host:       "omega",
		Venv:       "venv",
		Node:       "v22.0.0",
		Go:         "go1.22.0",
		Battery:    "87%",
		Jobs:       1,
	}
}

func contextFromPreview(ctx PreviewContext) *fakeContext {
	return &fakeContext{
		Username:   ctx.Username,
		Cwd:        ctx.Cwd,
		GitBranch:  ctx.GitBranch,
		GitDirty:   ctx.GitDirty,
		ExitStatus: ctx.ExitStatus,
		Host:       ctx.Host,
		Venv:       ctx.Venv,
		Node:       ctx.Node,
		Go:         ctx.Go,
		Battery:    ctx.Battery,
		Jobs:       ctx.Jobs,
	}
}

func renderUser(cfg config.SegmentConfig, ctx *fakeContext) string {
	if !cfg.Enabled {
		return ""
	}
	return ctx.Username
}

func renderCwd(cfg config.SegmentConfig, ctx *fakeContext) string {
	if !cfg.Enabled {
		return ""
	}
	return ctx.Cwd
}

func renderGit(cfg config.SegmentConfig, ctx *fakeContext) string {
	if !cfg.Enabled {
		return ""
	}
	if ctx.GitBranch == "" {
		return ""
	}
	out := ctx.GitBranch
	if ctx.GitDirty {
		out += " +"
	}
	return out
}

func renderStatus(cfg config.SegmentConfig, ctx *fakeContext) string {
	if !cfg.Enabled {
		return ""
	}
	if ctx.ExitStatus == 0 && !cfg.ShowSuccess {
		return ""
	}
	if ctx.ExitStatus == 0 {
		return "✓"
	}
	return fmt.Sprintf("✘ %d", ctx.ExitStatus)
}

func renderTime(cfg config.SegmentConfig, ctx *fakeContext) string {
	if !cfg.Enabled {
		return ""
	}
	return "14:52"
}

func renderHost(cfg config.SegmentConfig, ctx *fakeContext) string {
	if !cfg.Enabled {
		return ""
	}
	return ctx.Host
}

func renderVenv(cfg config.SegmentConfig, ctx *fakeContext) string {
	if !cfg.Enabled {
		return ""
	}
	return ctx.Venv
}

func renderNode(cfg config.SegmentConfig, ctx *fakeContext) string {
	if !cfg.Enabled {
		return ""
	}
	return ctx.Node
}

func renderGo(cfg config.SegmentConfig, ctx *fakeContext) string {
	if !cfg.Enabled {
		return ""
	}
	return ctx.Go
}

func renderBattery(cfg config.SegmentConfig, ctx *fakeContext) string {
	if !cfg.Enabled {
		return ""
	}
	return ctx.Battery
}

func renderJobs(cfg config.SegmentConfig, ctx *fakeContext) string {
	if !cfg.Enabled || ctx.Jobs == 0 {
		return ""
	}
	return fmt.Sprintf("%d jobs", ctx.Jobs)
}
