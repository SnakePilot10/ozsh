package config

const CurrentConfigVersion = 2

const (
	IconModeCompatible = "compatible"
	IconModeNerd       = "nerd"

	PromptLayoutOneLine = "one-line"
	PromptLayoutTwoLine = "two-line"
)

type Config struct {
	Version int          `toml:"version"`
	Prompt  PromptConfig `toml:"prompt"`
	Plugins PluginConfig `toml:"plugins"`
	Theme   ThemeConfig  `toml:"theme"`
}

type PromptConfig struct {
	Style                string                   `toml:"style"`
	DisplayName          string                   `toml:"display_name"`
	IconMode             string                   `toml:"icon_mode"`
	Layout               string                   `toml:"layout"`
	Symbol               string                   `toml:"symbol"`
	Newline              bool                     `toml:"newline"`
	RightPrompt          bool                     `toml:"right_prompt"`
	DisableHeavySegments bool                     `toml:"disable_heavy_segments"`
	Separator            string                   `toml:"separator"`
	Order                []string                 `toml:"order"`
	RightOrder           []string                 `toml:"right_order"`
	Segments             map[string]SegmentConfig `toml:"segments"`
}

type SegmentConfig struct {
	Enabled        bool   `toml:"enabled"`
	Icon           string `toml:"icon,omitempty"`
	CompatibleIcon string `toml:"compatible_icon"`
	NerdIcon       string `toml:"nerd_icon"`
	FG             string `toml:"fg"`
	BG             string `toml:"bg"`
	Bold           bool   `toml:"bold"`
	ShowSuccess    bool   `toml:"show_success,omitempty"`
}

type PluginConfig struct {
	Engine   string       `toml:"engine"`
	Selected []string     `toml:"selected"`
	Items    []PluginItem `toml:"items"`
}

type PluginItem struct {
	Name    string `toml:"name"`
	Enabled bool   `toml:"enabled"`
	Trusted bool   `toml:"trusted"`
	Source  string `toml:"source"`
	Load    string `toml:"load"`
}

type ThemeConfig struct {
	ID         string `toml:"id"`
	Variant    string `toml:"variant"`
	Name       string `toml:"name"`
	Accent     string `toml:"accent"`
	Background string `toml:"background"`
	Muted      string `toml:"muted"`
	Success    string `toml:"success"`
	Warning    string `toml:"warning"`
	Error      string `toml:"error"`
}
