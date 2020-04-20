package config

// Config describes config for SCIM.
type ScimConfig struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
	Password string `json:"password"`
}
