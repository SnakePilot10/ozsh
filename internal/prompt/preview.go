package prompt

import (
    "fmt"
    "strconv"
    "strings"

    "github.com/snakepilot10/ozsh/internal/config"
)

// Simulated returns a simulated prompt preview using default preview context.
// It clones the provided config and fills defaults to avoid mutating the caller's config.
func Simulated(cfg *config.Config) string {
    clone := *cfg
    config.FillDefaults(&clone)
    return SimulatedWithContext(&clone, DefaultPreviewContext())
}

// SimulatedWithContext returns a simulated prompt preview using the given preview context.
// It clones the provided config and fills defaults to avoid mutating the caller's config.
func SimulatedWithContext(cfg *config.Config, previewCtx PreviewContext) string {
    clone := *cfg
    config.FillDefaults(&clone)

    ctx := contextFromPreview(previewCtx)
    var parts []string
    var rightParts []string

    for _, name := range clone.Prompt.Order {
        rendered := renderPreviewSegment(&clone, ctx, name)
        if rendered != "" {
            parts = append(parts, rendered)
        }
    }
    for _, name := range clone.Prompt.RightOrder {
        rendered := renderPreviewSegment(&clone, ctx, name)
        if rendered != "" {
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
    return ansiWrap(rendered, segCfg)
}

func ansiWrap(value string, cfg config.SegmentConfig) string {
    open := ansiFG(cfg.FG)
    if cfg.Bold {
        open += "\x1b[1m"
    }
    if open == "" {
        return value
    }
    return open + value + "\x1b[0m"
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
