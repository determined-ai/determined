package logger

import "github.com/sirupsen/logrus"

type Context logrus.Fields

func (c Context) Fields() logrus.Fields {
	return logrus.Fields(c)
}

// MergeContexts returns a new merged Context object from the inputs, prefering later inputs.
func MergeContexts(xs ...Context) Context {
	ys := Context{}
	for _, x := range xs {
		for k, v := range x {
			ys[k] = v
		}
	}
	return ys
}
