package config

var Presets = map[string]ThemeConfig{
	"cyber-cyan": {
		Name:       "cyber-cyan",
		Accent:     "#00f5ff",
		Background: "#09090d",
		Muted:      "#6b6b80",
		Success:    "#00ff9f",
		Warning:    "#ffe600",
		Error:      "#ff003c",
	},
	"neon-red": {
		Name:       "neon-red",
		Accent:     "#ff003c",
		Background: "#09090d",
		Muted:      "#6b6b80",
		Success:    "#00ff9f",
		Warning:    "#ffe600",
		Error:      "#ff003c",
	},
	"matrix-green": {
		Name:       "matrix-green",
		Accent:     "#00ff9f",
		Background: "#050807",
		Muted:      "#6b806f",
		Success:    "#00ff9f",
		Warning:    "#ffe600",
		Error:      "#ff003c",
	},
}
