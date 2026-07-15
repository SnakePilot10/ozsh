package config

func Default() *Config {
	return &Config{
		Prompt: PromptConfig{
			Style:                "simple",
			Newline:              true,
			RightPrompt:          false,
			DisableHeavySegments: false,
			Separator:            "  ",
			Order:                []string{"user", "cwd", "git", "status"},
			RightOrder:           []string{},
			Segments: map[string]SegmentConfig{
				"user": {
					Enabled: true,
					Icon:    "",
					FG:      "cyan",
					BG:      "",
					Bold:    true,
				},
				"cwd": {
					Enabled: true,
					Icon:    "",
					FG:      "yellow",
					BG:      "",
					Bold:    false,
				},
				"git": {
					Enabled: true,
					Icon:    "",
					FG:      "magenta",
					BG:      "",
					Bold:    false,
				},
				"status": {
					Enabled:     true,
					Icon:        "",
					FG:          "red",
					BG:          "",
					Bold:        true,
					ShowSuccess: false,
				},
				"time": {
					Enabled: false,
					Icon:    "",
					FG:      "blue",
					BG:      "",
					Bold:    false,
				},
				"host": {
					Enabled: false,
					Icon:    "",
					FG:      "cyan",
					BG:      "",
					Bold:    false,
				},
				"venv": {
					Enabled: false,
					Icon:    "",
					FG:      "green",
					BG:      "",
					Bold:    false,
				},
				"node": {
					Enabled: false,
					Icon:    "",
					FG:      "green",
					BG:      "",
					Bold:    false,
				},
				"go": {
					Enabled: false,
					Icon:    "",
					FG:      "cyan",
					BG:      "",
					Bold:    false,
				},
				"battery": {
					Enabled: false,
					Icon:    "",
					FG:      "yellow",
					BG:      "",
					Bold:    false,
				},
				"jobs": {
					Enabled: false,
					Icon:    "",
					FG:      "magenta",
					BG:      "",
					Bold:    false,
				},
			},
		},
		Plugins: PluginConfig{
			Engine: "manual",
			Items:  []PluginItem{},
		},
		Theme: ThemeConfig{
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
