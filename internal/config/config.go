package config

type Config struct {
	Prompt  PromptConfig `toml:"prompt"`
	Plugins PluginConfig `toml:"plugins"`
	Theme   ThemeConfig  `toml:"theme"`
	Header  HeaderConfig `toml:"header"`
}

type PromptConfig struct {
	Style                string                   `toml:"style"`
	Newline              bool                     `toml:"newline"`
	RightPrompt          bool                     `toml:"right_prompt"`
	DisableHeavySegments bool                     `toml:"disable_heavy_segments"`
	Separator            string                   `toml:"separator"`
	Order                []string                 `toml:"order"`
	RightOrder           []string                 `toml:"right_order"`
	Segments             map[string]SegmentConfig `toml:"segments"`
}

type SegmentConfig struct {
	Enabled     bool   `toml:"enabled"`
	Icon        string `toml:"icon"`
	FG          string `toml:"fg"`
	BG          string `toml:"bg"`
	Bold        bool   `toml:"bold"`
	ShowSuccess bool   `toml:"show_success,omitempty"`
}

type PluginConfig struct {
	Engine string       `toml:"engine"`
	Items  []PluginItem `toml:"items"`
}

type PluginItem struct {
	Name    string `toml:"name"`
	Enabled bool   `toml:"enabled"`
	Trusted bool   `toml:"trusted"`
	Source  string `toml:"source"`
	Load    string `toml:"load"`
}

type ThemeConfig struct {
	Name       string `toml:"name"`
	Accent     string `toml:"accent"`
	Background string `toml:"background"`
	Muted      string `toml:"muted"`
	Success    string `toml:"success"`
	Warning    string `toml:"warning"`
	Error      string `toml:"error"`
}

type HeaderConfig struct {
	Enabled bool   `toml:"enabled"`
	Style   string `toml:"style"`
	Text    string `toml:"text"`
}
