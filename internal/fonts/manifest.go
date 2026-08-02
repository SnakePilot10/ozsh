package fonts

type Font struct {
	ID          string
	Name        string
	Version     string
	AssetURL    string
	SHA256      string
	ArchivePath string
	Recommended bool
}

var pinnedManifest = []Font{
	{
		ID: "jetbrains-mono", Name: "JetBrainsMono Nerd Font", Version: "v3.4.0",
		AssetURL:    "https://github.com/ryanoasis/nerd-fonts/releases/download/v3.4.0/JetBrainsMono.zip",
		SHA256:      "76f05ff3ace48a464a6ca57977998784ff7bdbb65a6d915d7e401cd3927c493c",
		ArchivePath: "JetBrainsMonoNerdFontMono-Regular.ttf", Recommended: true,
	},
	{
		ID: "meslo-lgs", Name: "MesloLGS Nerd Font", Version: "v3.4.0",
		AssetURL:    "https://github.com/ryanoasis/nerd-fonts/releases/download/v3.4.0/Meslo.zip",
		SHA256:      "13b502ac8c2bd9d3161018064560e23cd42b175bb730780a270975265a19ad57",
		ArchivePath: "MesloLGSNerdFontMono-Regular.ttf",
	},
	{
		ID: "fira-code", Name: "FiraCode Nerd Font", Version: "v3.4.0",
		AssetURL:    "https://github.com/ryanoasis/nerd-fonts/releases/download/v3.4.0/FiraCode.zip",
		SHA256:      "7cc4ffd8f7a1fc914cdab7b149808298165ff7a7f40e40d82dea9ebe41e8ca0b",
		ArchivePath: "FiraCodeNerdFontMono-Regular.ttf",
	},
}

func Manifest() []Font {
	return append([]Font(nil), pinnedManifest...)
}
