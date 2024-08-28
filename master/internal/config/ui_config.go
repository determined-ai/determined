package config

import (
	"os"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/license"
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
	const (
		orientationHorizontal = "horizontal"
		orientationVertical   = "vertical"
	)
	switch mode {
	case "dark":
		switch orientation {
		case orientationVertical:
			return m.DarkVeritical
		default:
			return m.DarkHorizontal
		}
	default:
		switch orientation {
		case orientationVertical:
			return m.LightVeritical
		default:
			return m.LightHorizontal
		}
	}
}

// UICustomizationConfig holds the configuration for customizing the UI.
type UICustomizationConfig struct {
	// LogoPaths is the path to variations of the custom logo to use in the web UI.
	LogoPaths *MediaAssetVariations `json:"logo_paths"`
}

// Validate checks if the paths in UICustomizationConfig are valid filesystem paths and reachable.
func (u UICustomizationConfig) Validate() []error {
	var errs []error
	if u.LogoPaths == nil {
		return errs
	}

	paths := map[string]string{
		"LightHorizontal": u.LogoPaths.LightHorizontal,
		"LightVeritical":  u.LogoPaths.LightVeritical,
		"DarkHorizontal":  u.LogoPaths.DarkHorizontal,
		"DarkVeritical":   u.LogoPaths.DarkVeritical,
	}

	for name, path := range paths {
		if path == "" {
			errs = append(errs, errors.New(name+" path is not set"))
			continue
		}
		license.RequireLicense("UI Customization")
		info, err := os.Stat(path)
		switch {
		case os.IsNotExist(err):
			errs = append(errs, errors.New(name+" path is not reachable: "+path))
		case err != nil:
			errs = append(errs, errors.New(name+" path error: "+err.Error()))
		case info.IsDir():
			errs = append(errs, errors.New(name+" path is a directory, not a file: "+path))
		}
	}

	return errs
}

// HasCustomLogo returns whether the UI customization has a custom logo.
func (u UICustomizationConfig) HasCustomLogo() bool {
	return u.LogoPaths != nil
}
