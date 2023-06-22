package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricIdentifierSerialize(t *testing.T) {
	type fields struct {
		Type MetricType
		Name string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		// TODO
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MetricIdentifier{
				Type: tt.fields.Type,
				Name: tt.fields.Name,
			}
			got, err := m.Serialize()
			if (err != nil) != tt.wantErr {
				t.Errorf("MetricIdentifier.Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MetricIdentifier.Serialize() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
	for _, tt := range tests {
		t.Run("deserialize metric identifier", func(t *testing.T) {
			got, err := DeserializeMetricIdentifier(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeserializeMetricIdentifier(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMetricIdentifierTwoWay(t *testing.T) {
	metricType := MetricType("ValidationMetricType")
	metricName := "ValidationMetricName"
	metricID := MetricIdentifier{Type: metricType, Name: metricName}
	s, err := metricID.Serialize()
	if err != nil {
		t.Errorf("Serialize error = %v", err)
	}
	metricID2, err := DeserializeMetricIdentifier(s)
	if err != nil {
		t.Errorf("Deserialize error = %v", err)
	}
	if metricID2.Type != metricID.Type || metricID2.Name != metricID.Name {
		t.Errorf("MetricIdentifierTwoWay error = %v", err)
	}
}
