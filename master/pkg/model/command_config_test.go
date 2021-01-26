package model

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestConfigValidate(t *testing.T) {
	type fields struct {
		Description string
		BindMounts  expconf.BindMountsConfig
		Environment expconf.EnvironmentConfig
		Resources   ResourcesConfig
		Entrypoint  []string
	}
	type testCase struct {
		name    string
		fields  fields
		wantErr bool
	}
	var environment expconf.EnvironmentConfig
	resources := ResourcesConfig{
		Slots:  1,
		Weight: 1,
	}

	tests := []testCase{
		{
			name: "valid",
			fields: fields{
				Resources:   resources,
				Environment: environment,
				Entrypoint: []string{
					"test",
				},
			},
		},
		{
			name: "invalid",
			fields: fields{
				Resources:   resources,
				Environment: environment,
			},
			wantErr: true,
		},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			c := &CommandConfig{
				Description: tc.fields.Description,
				BindMounts:  tc.fields.BindMounts,
				Environment: tc.fields.Environment,
				Resources:   tc.fields.Resources,
				Entrypoint:  tc.fields.Entrypoint,
			}
			if err := check.Validate(c); (err != nil) != tc.wantErr {
				t.Errorf("config.Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}
