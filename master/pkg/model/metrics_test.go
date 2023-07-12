package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricIdentifierDeserialize(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		args    args
		want    *MetricIdentifier
		wantErr bool
	}{
		{
			args{s: "ValidationMetricType.ValidationMetricName"},
			&MetricIdentifier{Type: "ValidationMetricType", Name: "ValidationMetricName"},
			false,
		},
		{
			args{s: "TrainingMetricType.TrainingMetricName"},
			&MetricIdentifier{Type: "TrainingMetricType", Name: "TrainingMetricName"},
			false,
		},
		{
			args{s: ""},
			nil,
			true,
		},
		{
			args{s: "ValidationMetricType"},
			nil,
			true,
		},
		{
			args{s: ".ValidationMetricName"},
			nil,
			true,
		},
		{
			args{s: ".."},
			nil,
			true,
		},
		{
			args{s: "."},
			nil,
			true,
		},
		{
			args{s: "ValidationMetricType."},
			nil,
			true,
		},
		{
			args{s: "ValidationMetricType.ValidationMetricName.Extra"},
			&MetricIdentifier{Type: "ValidationMetricType", Name: "ValidationMetricName.Extra"},
			false,
		},
	}
	for idx, tt := range tests {
		t.Run(fmt.Sprint(idx), func(t *testing.T) {
			got, err := DeserializeMetricIdentifier(tt.args.s)
			if tt.wantErr {
				require.Error(t, err, "Expected error with arg %v", tt.args.s)
			} else {
				require.NoError(t, err, "Unexpected error with arg %v", tt.args.s)
			}
			require.Equal(t, tt.want, got)
		})
	}
}
