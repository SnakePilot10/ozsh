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
				"user":    {Enabled: true, CompatibleIcon: "@", NerdIcon: "", FG: "cyan", Bold: true},
				"cwd":     {Enabled: true, CompatibleIcon: ">", NerdIcon: "", FG: "yellow"},
				"git":     {Enabled: true, CompatibleIcon: "git", NerdIcon: "", FG: "magenta"},
				"status":  {Enabled: true, CompatibleIcon: "!", NerdIcon: "", FG: "red", Bold: true, ShowSuccess: false},
				"time":    {Enabled: false, CompatibleIcon: ":", NerdIcon: "", FG: "blue"},
				"host":    {Enabled: false, CompatibleIcon: "#", NerdIcon: "󰒋", FG: "cyan"},
				"venv":    {Enabled: false, CompatibleIcon: "py", NerdIcon: "", FG: "green"},
				"node":    {Enabled: false, CompatibleIcon: "node", NerdIcon: "", FG: "green"},
				"go":      {Enabled: false, CompatibleIcon: "go", NerdIcon: "", FG: "cyan"},
				"battery": {Enabled: false, CompatibleIcon: "%", NerdIcon: "", FG: "yellow"},
				"jobs":    {Enabled: false, CompatibleIcon: "&", NerdIcon: "", FG: "magenta"},
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
			Name:       "Cyberpunk",
			Accent:     "#00f5ff",
			Background: "#09090d",
			Muted:      "#6b6b80",
			Success:    "#00ff9f",
			Warning:    "#ffe600",
			Error:      "#ff003c",
		},
	}
}
