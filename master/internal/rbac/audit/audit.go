package audit

import (
	"context"
	"github.com/sirupsen/logrus"
)

func ExtractLogFields(ctx context.Context) logrus.Fields {
	fields := ctx.Value("logFields")
	if fields == nil {
		return logrus.Fields{}
	}
	logFields, ok := fields.(logrus.Fields)
	if !ok {
		return logrus.Fields{}
	}
	return logFields
}
