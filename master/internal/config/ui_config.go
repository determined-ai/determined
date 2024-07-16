package config

import (
	"os"

	"github.com/pkg/errors"
)

// MediaAssetVariations allow variations of a media asset to be defined.
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
			if m.LightHorizontal != "" {
				return m.LightHorizontal
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
			if m.DarkHorizontal != "" {
				return m.DarkHorizontal
			}
		}
	}

	return m.LightHorizontal
}

type UICustomizationConfig struct {
	// LogoPath is the path to variation of custom logo to use in the web UI.
	LogoPath MediaAssetVariations `json:"logo_path"`
}

// Validate checks if the paths in UICustomizationConfig are valid filesystem paths and reachable.
func (u UICustomizationConfig) Validate() []error {
	var errs []error

	paths := map[string]string{
		"LightHorizontal": u.LogoPath.LightHorizontal,
		"LightVeritical":  u.LogoPath.LightVeritical,
		"DarkHorizontal":  u.LogoPath.DarkHorizontal,
		"DarkVeritical":   u.LogoPath.DarkVeritical,
	}

	for name, path := range paths {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			errs = append(errs, errors.New(name+" path is not reachable: "+path))
		}
	}

	return errs
}

// HasCustomLogo returns whether the UI customization has a custom logo.
func (u UICustomizationConfig) HasCustomLogo() bool {
	// If one exists, we're good
	return u.LogoPath.LightHorizontal != "" || u.LogoPath.LightVeritical != "" ||
		u.LogoPath.DarkHorizontal != "" || u.LogoPath.DarkVeritical != ""
}
