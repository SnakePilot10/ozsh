package config

var HeaderPresets = map[string]HeaderConfig{
	"ozsh":   {Enabled: true, Style: "ascii", Text: "ozsh"},
	"snake":  {Enabled: true, Style: "ascii", Text: "snake"},
	"omega":  {Enabled: true, Style: "ascii", Text: "omega"},
	"neon":   {Enabled: true, Style: "ascii", Text: "neon"},
	"matrix": {Enabled: true, Style: "ascii", Text: "matrix"},
	"termux": {Enabled: true, Style: "ascii", Text: "termux"},
}
