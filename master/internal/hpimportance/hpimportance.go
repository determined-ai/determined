package hpimportance

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

func computeHPImportance(data []model.HPImportanceTrialData) map[string]float64 {
	// This is a place-holder for a future Random Forest implementation to actually compute the
	// importance of each hyperparameter.
	output := make(map[string]float64)
	for key := range data[0].Hparams {
		output[key] = 0.0
	}
	return output
}
