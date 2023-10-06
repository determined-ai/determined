package model

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestTaskLogLevelFromLogrus(t *testing.T) {
	for _, l := range logrus.AllLevels {
		require.NotEqual(t, LogLevelUnspecified, TaskLogLevelFromLogrus(l), "unhandled log level")
	}
}
