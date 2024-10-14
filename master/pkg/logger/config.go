package logger

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/config"
)

// SetLogrus sets logrus globally.
func SetLogrus(c config.LoggerConfig) {
	level, err := logrus.ParseLevel(c.Level)
	if err != nil {
		panic(fmt.Sprintf("invalid log level: %s", c.Level))
	}

	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
		DisableColors: !c.Color,
	})
}
