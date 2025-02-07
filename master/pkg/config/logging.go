package config

import "github.com/sirupsen/logrus"

// LoggerConfig is the configuration of logger.
type LoggerConfig struct {
	Level string `json:"level"`
	Color bool   `json:"color"`
}

// Validate implements the check.Validatable interface.
func (c LoggerConfig) Validate() []error {
	if _, err := logrus.ParseLevel(c.Level); err != nil {
		return []error{err}
	}
	return nil
}

// DefaultLoggerConfig returns the default configuration of logger.
func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Level: "info",
		Color: true,
	}
}
