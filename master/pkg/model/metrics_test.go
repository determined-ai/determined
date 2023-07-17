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
			args{s: "ValidationMetricGroup.ValidationMetricName"},
			&MetricIdentifier{Group: "ValidationMetricGroup", Name: "ValidationMetricName"},
			false,
		},
		{
			args{s: "TrainingMetricGroup.TrainingMetricName"},
			&MetricIdentifier{Group: "TrainingMetricGroup", Name: "TrainingMetricName"},
			false,
		},
		{
			args{s: ""},
			nil,
			true,
		},
		{
			args{s: "ValidationMetricGroup"},
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
			args{s: "ValidationMetricGroup."},
			nil,
			true,
		},
		{
			args{s: "ValidationMetricGroup.ValidationMetricName.Extra"},
			&MetricIdentifier{Group: "ValidationMetricGroup", Name: "ValidationMetricName.Extra"},
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
