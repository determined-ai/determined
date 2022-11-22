package logger

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// DefaultConfig returns the default configuration of logger.
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Color:      true,
		Structured: true,
	}
}

// Config is the configuration of logger.
type Config struct {
	Level      string `json:"level"`
	Color      bool   `json:"color"`
	Structured bool   `json:"structured"`
}

// Validate implements the check.Validatable interface.
func (c Config) Validate() []error {
	if _, err := logrus.ParseLevel(c.Level); err != nil {
		return []error{err}
	}
	return nil
}

// SetLogrus sets logrus globally.
func SetLogrus(c Config) {
	level, err := logrus.ParseLevel(c.Level)
	if err != nil {
		panic(fmt.Sprintf("invalid log level: %s", c.Level))
	}

	logrus.SetLevel(level)

	if !c.Structured {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
			DisableColors: !c.Color,
		})
	} else {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
}
