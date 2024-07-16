package config

// MediaAssetVariations allow varitions of a media asset to be defined.
type MediaAssetVariations struct {
	DarkHorizontal  string `json:"dark_horizontal"`
	DarkVeritical   string `json:"dark_vertical"`
	LightHorizontal string `json:"light_horizontal"`
	LightVeritical  string `json:"light_vertical"`
}

// PickVariation returns the best variation for the given mode and orientation.
func (m MediaAssetVariations) PickVariation(mode, orientation string) string {
	if mode == "" || mode == "light" {
		if orientation == "" || orientation == "horizontal" {
			if m.LightHorizontal != "" {
				return m.LightHorizontal
			}
		}
		if orientation == "" || orientation == "vertical" {
			if m.LightVeritical != "" {
				return m.LightVeritical
			}
		}
	}

	if mode == "dark" {
		if orientation == "" || orientation == "horizontal" {
			if m.DarkHorizontal != "" {
				return m.DarkHorizontal
			}
		}
		if orientation == "" || orientation == "vertical" {
			if m.DarkVeritical != "" {
				return m.DarkVeritical
			}
		}
	}

	return m.LightHorizontal
}

type UICustomizationConfig struct {
	LogoPath MediaAssetVariations `json:"logo_path"`
}

func (u UICustomizationConfig) Validate() []error {
	return nil
}

// HasCustomLogo returns whether the UI customization has a custom logo.
func (u UICustomizationConfig) HasCustomLogo() bool {
	// TODO.
	return u.LogoPath.LightHorizontal != ""
}
