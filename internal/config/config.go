package config

const CurrentConfigVersion = 2

type Config struct {
	Version int          `toml:"version"`
	Prompt  PromptConfig `toml:"prompt"`
	Plugins PluginConfig `toml:"plugins"`
	Theme   ThemeConfig  `toml:"theme"`
}

type PromptConfig struct {
	Style                string                   `toml:"style"`
	Newline              bool                     `toml:"newline"`
	RightPrompt          bool                     `toml:"right_prompt"`
	TransientPrompt      bool                     `toml:"transient_prompt"`
	OSC7                 bool                     `toml:"osc7"`
	OSC133               bool                     `toml:"osc133"`
	DisableHeavySegments bool                     `toml:"disable_heavy_segments"`
	Separator            string                   `toml:"separator"`
	Order                []string                 `toml:"order"`
	RightOrder           []string                 `toml:"right_order"`
	TransientOrder       []string                 `toml:"transient_order"`
	Segments             map[string]SegmentConfig `toml:"segments"`
}

type SegmentConfig struct {
	Enabled        bool   `toml:"enabled"`
	Icon           string `toml:"icon"`
	FG             string `toml:"fg"`
	BG             string `toml:"bg"`
	Bold           bool   `toml:"bold"`
	Italic         bool   `toml:"italic"`
	Underline      bool   `toml:"underline"`
	PaddingLeft    int    `toml:"padding_left"`
	PaddingRight   int    `toml:"padding_right"`
	LeadingSymbol  string `toml:"leading_symbol"`
	TrailingSymbol string `toml:"trailing_symbol"`
	When           string `toml:"when"`
	WhenEnv        string `toml:"when_env"`
	CacheTTL       int    `toml:"cache_ttl_seconds"`
	ShowSuccess    bool   `toml:"show_success,omitempty"`
}

type PluginConfig struct {
	Engine string       `toml:"engine"`
	Items  []PluginItem `toml:"items"`
}

type PluginItem struct {
	Name        string `toml:"name"`
	Enabled     bool   `toml:"enabled"`
	Trusted     bool   `toml:"trusted"`
	Source      string `toml:"source"`
	Load        string `toml:"load"`
	Repository  string `toml:"repository"`
	Revision    string `toml:"revision"`
	InstalledAt string `toml:"installed_at"`
}

type ThemeConfig struct {
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Author      string   `toml:"author"`
	Tier        string   `toml:"tier"`
	Requires    []string `toml:"requires"`
	Accent      string   `toml:"accent"`
	Background  string   `toml:"background"`
	Muted       string   `toml:"muted"`
	Success     string   `toml:"success"`
	Warning     string   `toml:"warning"`
	Error       string   `toml:"error"`
}
