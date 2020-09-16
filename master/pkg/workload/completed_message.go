package workload

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// CompletedMessage is the wrapping message returned by the trial runner when a workload
// is completed.
type CompletedMessage struct {
	Type              string          `json:"type"`
	Workload          Workload        `json:"workload"`
	StartTime         time.Time       `json:"start_time"`
	EndTime           time.Time       `json:"end_time"`
	ExitedReason      *ExitedReason   `json:"exited_reason"`
	RawMetrics        json.RawMessage `json:"metrics,omitempty"`
	CheckpointMetrics *CheckpointMetrics
	ValidationMetrics *ValidationMetrics
	RunMetrics        map[string]interface{}
}

// UnmarshalJSON unmarshals the provided bytes into a workload.CompletedMessage. An error is
// returned if the bytes cannot be unmarshaled.
func (message *CompletedMessage) UnmarshalJSON(bytes []byte) error {
	type base *CompletedMessage
	if err := json.Unmarshal(bytes, (base)(message)); err != nil {
		return err
	}

	// This check is needed for master restart: the metrics of RunStep completion messages are deleted
	// before they are stored in the searcher events table.
	if len(message.RawMetrics) == 0 {
		return nil
	}

	switch message.Workload.Kind {
	case RunStep:
		return json.Unmarshal(message.RawMetrics, &message.RunMetrics)
	case CheckpointModel:
		return json.Unmarshal(message.RawMetrics, &message.CheckpointMetrics)
	case ComputeValidationMetrics:
		return json.Unmarshal(message.RawMetrics, &message.ValidationMetrics)
	default:
		return errors.Errorf("unexpected workload kind unmarshaling: %s", message.Workload)
	}
}

// CheckpointMetrics contains the checkpoint metadata returned by the StorageManager after
// completing a checkpoint.
type CheckpointMetrics struct {
	UUID      uuid.UUID      `json:"uuid"`
	Resources map[string]int `json:"resources"`
	Framework string         `json:"framework"`
	Format    string         `json:"format"`
}

// ValidationMetrics contains the user-defined metrics calculated after a validation
// workload.
type ValidationMetrics struct {
	NumInputs int                    `json:"num_inputs"`
	Metrics   map[string]interface{} `json:"validation_metrics"`
}

// Metric returns the requested validation metric value from the set of validation metrics.
func (metrics ValidationMetrics) Metric(name string) (float64, error) {
	rawMetric, ok := metrics.Metrics[name]
	if !ok {
		return 0, errors.Errorf("'%s' could not be found in validation metrics", name)
	}
	metric, ok := rawMetric.(float64)
	if !ok {
		return 0, errors.Errorf("'%s' is not a scalar float value", name)
	}
	return metric, nil
}

// ExitedReason defines why a workload exited early.
type ExitedReason string

const (
	// Errored signals the searcher that the workload errored out.
	Errored ExitedReason = "ERRORED"
	// UserCanceled signals the searcher that the user requested a cancelation.
	UserCanceled ExitedReason = "USER_CANCELED"
)
