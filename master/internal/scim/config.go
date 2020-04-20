package scim

// Config describes config for SCIM.
type Config struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
	Password string `json:"password"`
}
