package db

// DefaultConfig returns the default configuration of the database.
func DefaultConfig() *Config {
	return &Config{
		Migrations: "file://static/migrations",
	}
}

// Config hosts configuration fields of the database.
type Config struct {
	User       string `json:"user"`
	Password   string `json:"password"`
	Migrations string `json:"migrations"`
	Host       string `json:"host"`
	Port       string `json:"port"`
	Name       string `json:"name"`
}
