package prompt

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/snakepilot10/ozsh/internal/config"
)

func Simulated(cfg *config.Config) string {
	return SimulatedWithContext(cfg, DefaultPreviewContext())
}

func SimulatedWithContext(cfg *config.Config, previewCtx PreviewContext) string {
	clone := cloneForPreview(cfg)
	config.FillDefaults(&clone)

	ctx := contextFromPreview(previewCtx)
	parts := make([]string, 0, len(clone.Prompt.Order))
	rightParts := make([]string, 0, len(clone.Prompt.RightOrder))

	for _, name := range clone.Prompt.Order {
		if rendered := renderPreviewSegment(&clone, ctx, name); rendered != "" {
			parts = append(parts, rendered)
		}
	}
	for _, name := range clone.Prompt.RightOrder {
		if rendered := renderPreviewSegment(&clone, ctx, name); rendered != "" {
			rightParts = append(rightParts, rendered)
		}
	}

	line := strings.Join(parts, clone.Prompt.Separator)
	if clone.Prompt.RightPrompt || len(rightParts) > 0 {
		line = fmt.Sprintf("%s    %s", line, strings.Join(rightParts, clone.Prompt.Separator))
	}
	if clone.Prompt.Newline {
		return fmt.Sprintf("%s\n❯", line)
	}
	return fmt.Sprintf("%s ❯", line)
}

func cloneForPreview(cfg *config.Config) config.Config {
	clone := *cfg
	clone.Prompt.Order = append([]string(nil), cfg.Prompt.Order...)
	clone.Prompt.RightOrder = append([]string(nil), cfg.Prompt.RightOrder...)
	clone.Prompt.Segments = make(map[string]config.SegmentConfig, len(cfg.Prompt.Segments))
	for name, segment := range cfg.Prompt.Segments {
		clone.Prompt.Segments[name] = segment
	}
	clone.Plugins.Items = append([]config.PluginItem(nil), cfg.Plugins.Items...)
	return clone
}

func renderPreviewSegment(cfg *config.Config, ctx *fakeContext, name string) string {
	segCfg, ok := cfg.Prompt.Segments[name]
	if !ok || !segmentActive(cfg, name, segCfg) {
		return ""
	}
	fn, ok := segmentRegistry[name]
	if !ok {
		return ""
	}
	rendered := fn(segCfg, ctx)
	if rendered == "" {
		return ""
	}
	if segCfg.Icon != "" {
		rendered = segCfg.Icon + " " + rendered
	}
	return ansiWrap(rendered, segCfg)
}

func ansiWrap(value string, cfg config.SegmentConfig) string {
	open := ansiBG(cfg.BG) + ansiFG(cfg.FG)
	if cfg.Bold {
		open += "\x1b[1m"
	}
	if open == "" {
		return value
	}
	return open + value + "\x1b[0m"
}

func ansiBG(color string) string {
	if strings.HasPrefix(color, "#") && len(color) == 7 {
		r, errR := strconv.ParseInt(color[1:3], 16, 64)
		g, errG := strconv.ParseInt(color[3:5], 16, 64)
		b, errB := strconv.ParseInt(color[5:7], 16, 64)
		if errR == nil && errG == nil && errB == nil {
			return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
		}
	}
	colors := map[string]string{
		"black": "40", "red": "41", "green": "42", "yellow": "43",
		"blue": "44", "magenta": "45", "cyan": "46", "white": "47",
	}
	if code := colors[color]; code != "" {
		return "\x1b[" + code + "m"
	}
	return ""
}

func ansiFG(color string) string {
	if strings.HasPrefix(color, "#") && len(color) == 7 {
		r, errR := strconv.ParseInt(color[1:3], 16, 64)
		g, errG := strconv.ParseInt(color[3:5], 16, 64)
		b, errB := strconv.ParseInt(color[5:7], 16, 64)
		if errR == nil && errG == nil && errB == nil {
			return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
		}
	}
	switch color {
	case "black":
		return "\x1b[30m"
	case "red":
		return "\x1b[31m"
	case "green":
		return "\x1b[32m"
	case "yellow":
		return "\x1b[33m"
	case "blue":
		return "\x1b[34m"
	case "magenta":
		return "\x1b[35m"
	case "cyan":
		return "\x1b[36m"
	case "white":
		return "\x1b[37m"
	default:
		return ""
	}
}
