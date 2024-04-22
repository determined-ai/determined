package config

import (
	"net/url"
)

// DetCloudConfig allows det-cloud specific configuration.
type DetCloudConfig struct {
	Enabled  bool   `json:"enabled"`
	LoginURL string `json:"login_url"`
}

// Validate implements the check.Validatable interface.
func (c DetCloudConfig) Validate() []error {
	if !c.Enabled {
		return nil
	}

	_, err := url.Parse(c.LoginURL)
	return []error{err}
}
