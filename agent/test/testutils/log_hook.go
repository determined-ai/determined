//go:build integration

package testutils

import (
	"github.com/sirupsen/logrus"
)

// LogChannel is an channel implenting logrus.Hook.
type LogChannel chan *logrus.Entry

// NewLogChannel creates a new LogChannel.
func NewLogChannel(bufSize int) LogChannel {
	return make(chan *logrus.Entry, bufSize)
}

// Fire implements the logrus.Hook interface.
func (lc LogChannel) Fire(entry *logrus.Entry) error {
	lc <- entry
	return nil
}

// Levels implements the logrus.Hook interface.
func (lc LogChannel) Levels() []logrus.Level {
	return logrus.AllLevels
}
