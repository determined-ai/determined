package config

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/union"
)

// ScimConfig describes config for SCIM.
type ScimConfig struct {
	Enabled bool       `json:"enabled"`
	Auth    AuthConfig `json:"auth"`
}

// AuthConfig describes authentication configuration for SCIM.
type AuthConfig struct {
	BasicAuthConfig *BasicAuthConfig `union:"type,basic" json:"-"`
	OAuthConfig     *OAuthConfig     `union:"type,oauth" json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (a AuthConfig) MarshalJSON() ([]byte, error) {
	if a.BasicAuthConfig == nil && a.OAuthConfig == nil {
		return json.Marshal(nil)
	}
	return union.Marshal(a)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *AuthConfig) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, a); err != nil {
		return err
	}
	type DefaultParser *AuthConfig
	return errors.Wrap(json.Unmarshal(data, DefaultParser(a)), "failed to parse SCIM auth config")
}

// BasicAuthConfig describes HTTP Basic authentication configuration for SCIM.
type BasicAuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Validate implements the check.Validatable interface.
func (b BasicAuthConfig) Validate() []error {
	var errs []error
	if b.Username == "" {
		errs = append(errs, errors.New("username is missing"))
	}
	if b.Password == "" {
		errs = append(errs, errors.New("password is missing"))
	}
	return errs
}

// OAuthConfig describes OAuth configuration for SCIM (currently empty because we need a placeholder
// for the union type unmarshaling).
type OAuthConfig struct{}
