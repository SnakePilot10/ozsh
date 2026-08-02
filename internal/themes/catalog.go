package themes

import (
	_ "embed"
	"fmt"
	"regexp"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/snakepilot10/ozsh/internal/config"
)

//go:embed catalog.toml
var catalogData []byte

type Preset struct {
	ID          string
	Name        string
	Description string
	Variant     string
	Layout      string
	Symbol      string
	Separator   string
	Order       []string
	RightOrder  []string
	UserIcon    string
	CwdIcon     string
	GitIcon     string
	StatusIcon  string
	Theme       config.ThemeConfig
}

type rawCatalog struct {
	Presets []rawPreset `toml:"preset"`
}

type rawPreset struct {
	ID          string   `toml:"id"`
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Variant     string   `toml:"variant"`
	Accent      string   `toml:"accent"`
	Background  string   `toml:"background"`
	Muted       string   `toml:"muted"`
	Success     string   `toml:"success"`
	Warning     string   `toml:"warning"`
	Error       string   `toml:"error"`
	Layout      string   `toml:"layout"`
	Symbol      string   `toml:"symbol"`
	Separator   string   `toml:"separator"`
	Order       []string `toml:"order"`
	RightOrder  []string `toml:"right_order"`
	UserIcon    string   `toml:"user_icon"`
	CwdIcon     string   `toml:"cwd_icon"`
	GitIcon     string   `toml:"git_icon"`
	StatusIcon  string   `toml:"status_icon"`
}

var (
	loadOnce     sync.Once
	loaded       []Preset
	loadErr      error
	idPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

func loadCatalog() ([]Preset, error) {
	loadOnce.Do(func() {
		var decoded rawCatalog
		if err := toml.Unmarshal(catalogData, &decoded); err != nil {
			loadErr = fmt.Errorf("decode theme catalog: %w", err)
			return
		}
		seen := map[string]struct{}{}
		for _, raw := range decoded.Presets {
			key := raw.ID + "\x00" + raw.Variant
			if _, ok := seen[key]; ok {
				loadErr = fmt.Errorf("duplicate theme preset %q variant %q", raw.ID, raw.Variant)
				return
			}
			seen[key] = struct{}{}
			if err := validateRaw(raw); err != nil {
				loadErr = err
				return
			}
			loaded = append(loaded, Preset{
				ID: raw.ID, Name: raw.Name, Description: raw.Description, Variant: raw.Variant,
				Layout: raw.Layout, Symbol: raw.Symbol, Separator: raw.Separator,
				Order: append([]string(nil), raw.Order...), RightOrder: append([]string(nil), raw.RightOrder...),
				UserIcon: raw.UserIcon, CwdIcon: raw.CwdIcon, GitIcon: raw.GitIcon, StatusIcon: raw.StatusIcon,
				Theme: config.ThemeConfig{
					ID: raw.ID, Variant: raw.Variant, Name: raw.Name,
					Accent: raw.Accent, Background: raw.Background, Muted: raw.Muted,
					Success: raw.Success, Warning: raw.Warning, Error: raw.Error,
				},
			})
		}
		if len(loaded) == 0 {
			loadErr = fmt.Errorf("theme catalog is empty")
		}
	})
	return loaded, loadErr
}

func validateRaw(raw rawPreset) error {
	if !idPattern.MatchString(raw.ID) {
		return fmt.Errorf("invalid theme id %q", raw.ID)
	}
	if raw.Name == "" || raw.Description == "" || raw.Symbol == "" || raw.Separator == "" {
		return fmt.Errorf("theme %q has incomplete presentation metadata", raw.ID)
	}
	if raw.Layout != config.PromptLayoutOneLine && raw.Layout != config.PromptLayoutTwoLine {
		return fmt.Errorf("theme %q has invalid layout %q", raw.ID, raw.Layout)
	}
	for name, value := range map[string]string{
		"accent": raw.Accent, "background": raw.Background, "muted": raw.Muted,
		"success": raw.Success, "warning": raw.Warning, "error": raw.Error,
	} {
		if !colorPattern.MatchString(value) {
			return fmt.Errorf("theme %q has invalid %s color %q", raw.ID, name, value)
		}
	}
	if len(raw.Order) == 0 {
		return fmt.Errorf("theme %q has empty segment order", raw.ID)
	}
	return nil
}

// List returns every selectable preset. Theme families with variants, such as
// Circuit, appear once per variant so terminal users can choose them directly.
func List() []Preset {
	presets, err := loadCatalog()
	if err != nil {
		panic(err)
	}
	result := make([]Preset, 0, len(presets))
	for _, preset := range presets {
		result = append(result, clonePreset(preset))
	}
	return result
}

// Families returns one representative preset per theme ID. It is useful for
// compact CLI listings where variants are displayed underneath the family.
func Families() []Preset {
	presets, err := loadCatalog()
	if err != nil {
		panic(err)
	}
	seen := map[string]struct{}{}
	result := make([]Preset, 0, len(presets))
	for _, preset := range presets {
		if _, ok := seen[preset.ID]; ok {
			continue
		}
		seen[preset.ID] = struct{}{}
		result = append(result, clonePreset(preset))
	}
	return result
}

func Get(id, variant string) (Preset, bool) {
	presets, err := loadCatalog()
	if err != nil {
		return Preset{}, false
	}
	for _, preset := range presets {
		if preset.ID != id {
			continue
		}
		if variant == "" || preset.Variant == variant {
			return clonePreset(preset), true
		}
	}
	return Preset{}, false
}

func Variants(id string) []string {
	presets, err := loadCatalog()
	if err != nil {
		return nil
	}
	var variants []string
	for _, preset := range presets {
		if preset.ID == id && preset.Variant != "" {
			variants = append(variants, preset.Variant)
		}
	}
	return variants
}

func Apply(base *config.Config, preset Preset) *config.Config {
	cfg := cloneConfig(base)
	cfg.Theme = preset.Theme
	applyPresentation(cfg, preset)
	return cfg
}

func cloneConfig(source *config.Config) *config.Config {
	if source == nil {
		return config.Default()
	}
	clone := *source
	clone.Prompt.Order = append([]string(nil), source.Prompt.Order...)
	clone.Prompt.RightOrder = append([]string(nil), source.Prompt.RightOrder...)
	clone.Prompt.Segments = make(map[string]config.SegmentConfig, len(source.Prompt.Segments))
	for name, segment := range source.Prompt.Segments {
		clone.Prompt.Segments[name] = segment
	}
	clone.Plugins.Selected = append([]string(nil), source.Plugins.Selected...)
	clone.Plugins.Items = append([]config.PluginItem(nil), source.Plugins.Items...)
	return &clone
}

func clonePreset(preset Preset) Preset {
	preset.Order = append([]string(nil), preset.Order...)
	preset.RightOrder = append([]string(nil), preset.RightOrder...)
	return preset
}
