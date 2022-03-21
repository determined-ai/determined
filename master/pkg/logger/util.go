package logger

import "github.com/sirupsen/logrus"

// Context maintains context on logging fields to aid stitching together a history.
type Context logrus.Fields

// Fields returns the context's logrus.Fields to enrich a log.
func (c Context) Fields() logrus.Fields {
	return logrus.Fields(c)
}

// MergeContexts returns a new merged Context object from the inputs, preferring later inputs.
func MergeContexts(xs ...Context) Context {
	ys := Context{}
	for _, x := range xs {
		for k, v := range x {
			ys[k] = v
		}
	}
	return ys
}
