package fluent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func TestAdditionalOutputConfig(t *testing.T) {
	additionalFluentOutputs := `
[OUTPUT]
  Name file
  File test.log
  Match *
  Path /run/determined/fluent
`
	loggingConfigs := []model.LoggingConfig{
		{
			DefaultLoggingConfig: &model.DefaultLoggingConfig{
				AdditionalFluentOutputs: ptrs.Ptr(additionalFluentOutputs),
			},
		},
		{
			ElasticLoggingConfig: &model.ElasticLoggingConfig{
				Host: "test",
				Port: 9200,
				Security: model.ElasticSecurityConfig{
					Username: ptrs.Ptr("username"),
					Password: ptrs.Ptr("password"),
				},
				AdditionalFluentOutputs: ptrs.Ptr(additionalFluentOutputs),
			},
		},
	}

	for _, loggingConfig := range loggingConfigs {
		c := &strings.Builder{}
		makeOutputConfig(c, nil, "127.0.0.1", 8080, loggingConfig, model.TLSClientConfig{})
		require.Contains(t, c.String(), additionalFluentOutputs,
			"outputs did not contain additional fluent options")
	}
}
