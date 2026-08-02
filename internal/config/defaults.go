package config

var defaultSelectedPlugins = []string{
	"zsh-autosuggestions",
	"fzf-tab",
	"zsh-syntax-highlighting",
}

func Default() *Config {
	return &Config{
		Version: CurrentConfigVersion,
		Prompt: PromptConfig{
			Style:                "simple",
			DisplayName:          "",
			IconMode:             IconModeCompatible,
			Layout:               PromptLayoutTwoLine,
			Symbol:               "❯",
			Newline:              true,
			RightPrompt:          false,
			DisableHeavySegments: false,
			Separator:            "  ",
			Order:                []string{"user", "cwd", "git", "status"},
			RightOrder:           []string{},
			Segments: map[string]SegmentConfig{
				"user":    {Enabled: true, FG: "cyan", Bold: true},
				"cwd":     {Enabled: true, FG: "yellow"},
				"git":     {Enabled: true, FG: "magenta"},
				"status":  {Enabled: true, FG: "red", Bold: true, ShowSuccess: false},
				"time":    {Enabled: false, FG: "blue"},
				"host":    {Enabled: false, FG: "cyan"},
				"venv":    {Enabled: false, FG: "green"},
				"node":    {Enabled: false, FG: "green"},
				"go":      {Enabled: false, FG: "cyan"},
				"battery": {Enabled: false, FG: "yellow"},
				"jobs":    {Enabled: false, FG: "magenta"},
			},
		},
		Plugins: PluginConfig{
			Engine:   "manual",
			Selected: append([]string(nil), defaultSelectedPlugins...),
			Items:    []PluginItem{},
		},
		Theme: ThemeConfig{
			ID:         "cyberpunk",
			Variant:    "",
			Name:       "cyber-cyan",
			Accent:     "#00f5ff",
			Background: "#09090d",
			Muted:      "#6b6b80",
			Success:    "#00ff9f",
			Warning:    "#ffe600",
			Error:      "#ff003c",
		},
	}
}
