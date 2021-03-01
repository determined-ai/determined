package hpimportance

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestComputeHPImportance(t *testing.T) {
	masterConfig := HPImportanceConfig{
		WorkersLimit:   0,
		QueueLimit:     16,
		CoresPerWorker: 1,
		MaxTrees:       100,
	}
	expConfig := &model.ExperimentConfig{
		Hyperparameters: model.Hyperparameters{
			"dropout1": model.Hyperparameter{
				DoubleHyperparameter: &model.DoubleHyperparameter{
					Minval: 0.2,
					Maxval: 0.8,
				},
			},
			"dropout2": {
				DoubleHyperparameter: &model.DoubleHyperparameter{
					Minval: 0.2,
					Maxval: 0.8,
				},
			},
			"global_batch_size": {
				ConstHyperparameter: &model.ConstHyperparameter{
					Val: 64,
				},
			},
			"learning_rate": {
				DoubleHyperparameter: &model.DoubleHyperparameter{
					Minval: .0001,
					Maxval: 1.0,
				},
			},
			"n_filters1": {
				IntHyperparameter: &model.IntHyperparameter{
					Minval: 8,
					Maxval: 64,
				},
			},
			"n_filters2": {
				IntHyperparameter: &model.IntHyperparameter{
					Minval: 8,
					Maxval: 72,
				},
			},
			"n_filters3": {
				CategoricalHyperparameter: &model.CategoricalHyperparameter{
					Vals: []interface{}{"val1", "val2"},
				},
			},
		},
	}

	data := map[int][]model.HPImportanceTrialData{
		10: {
			{
				TrialID: 1,
				Hparams: map[string]interface{}{
					"dropout1":          0.57,
					"dropout2":          0.45,
					"global_batch_size": 64,
					"learning_rate":     0.5070331869253129,
					"n_filters1":        23,
					"n_filters2":        51,
					"n_filters3":        "val1",
				},
				Metric: 2.2962841987609863,
			},
			{
				TrialID: 2,
				Hparams: map[string]interface{}{
					"dropout1":          0.87,
					"dropout2":          0.23,
					"global_batch_size": 64,
					"learning_rate":     0.7506644734524253,
					"n_filters1":        34,
					"n_filters2":        45,
					"n_filters3":        "val1",
				},
				Metric: 2.2999706268310547,
			},
			{
				TrialID: 3,
				Hparams: map[string]interface{}{
					"dropout1":          0.87,
					"dropout2":          0.23,
					"global_batch_size": 64,
					"learning_rate":     0.3823986411952302,
					"n_filters1":        8,
					"n_filters2":        28,
					"n_filters3":        "val1",
				},
				Metric: 2.313760995864868,
			},
			{
				TrialID: 4,
				Hparams: map[string]interface{}{
					"dropout1":          0.87,
					"dropout2":          0.23,
					"global_batch_size": 64,
					"learning_rate":     0.9338878822452688,
					"n_filters1":        33,
					"n_filters2":        51,
					"n_filters3":        "val2",
				},
				Metric: 2.2808141708374023,
			},
			{
				TrialID: 5,
				Hparams: map[string]interface{}{
					"dropout1":          0.87,
					"dropout2":          0.23,
					"global_batch_size": 64,
					"learning_rate":     0.4746417441198819,
					"n_filters1":        39,
					"n_filters2":        13,
					"n_filters3":        "val2",
				},
				Metric: 2.2757034301757812,
			},
		},
		8: {
			{
				TrialID: 6,
				Hparams: map[string]interface{}{
					"dropout1":          0.87,
					"dropout2":          0.23,
					"global_batch_size": 64,
					"learning_rate":     0.34,
					"n_filters1":        23,
					"n_filters2":        51,
					"n_filters3":        "val1",
				},
				Metric: 2.2962841987609863,
			},
			{
				TrialID: 7,
				Hparams: map[string]interface{}{
					"dropout1":          0.87,
					"dropout2":          0.23,
					"global_batch_size": 64,
					"learning_rate":     0.089,
					"n_filters1":        19,
					"n_filters2":        30,
					"n_filters3":        "val1",
				},
				Metric: 2.2999706268310547,
			},
			{
				TrialID: 8,
				Hparams: map[string]interface{}{
					"dropout1":          0.87,
					"dropout2":          0.23,
					"global_batch_size": 64,
					"learning_rate":     0.089,
					"n_filters1":        19,
					"n_filters2":        30,
					"n_filters3":        "val1",
				},
				Metric: 2.2999706268310547,
			},
		},
	}
	nTreesResults, err := createDataFile(data, expConfig, ".")
	assert.NilError(t, err)
	assert.Equal(t, nTreesResults, 8)

	data[4] = []model.HPImportanceTrialData{
		{
			TrialID: 9,
			Hparams: map[string]interface{}{
				"dropout1":          0.1873934729034619,
				"dropout2":          0.3482374932749237,
				"global_batch_size": 64,
				"learning_rate":     0.3434324334322343,
				"n_filters1":        23,
				"n_filters2":        51,
				"n_filters3":        "val1",
			},
			Metric: 2.2962841987609863,
		},
		{
			TrialID: 10,
			Hparams: map[string]interface{}{
				"dropout1":          0.8742901348905638,
				"dropout2":          0.2348551036407937,
				"global_batch_size": 64,
				"learning_rate":     0.0893743468710934,
				"n_filters1":        19,
				"n_filters2":        30,
				"n_filters3":        "val1",
			},
			Metric: 2.2999706268310547,
		},
		{
			TrialID: 11,
			Hparams: map[string]interface{}{
				"dropout1":          0.8742901348905638,
				"dropout2":          0.2348551036407937,
				"global_batch_size": 64,
				"learning_rate":     0.0893743468710934,
				"n_filters1":        19,
				"n_filters2":        30,
				"n_filters3":        "val1",
			},
			Metric: 2.2999706268310547,
		},
	}

	data[20] = []model.HPImportanceTrialData{
		{
			TrialID: 12,
			Hparams: map[string]interface{}{
				"dropout1":          0.18,
				"dropout2":          0.34,
				"global_batch_size": 64,
				"learning_rate":     0.34,
				"n_filters1":        23,
				"n_filters2":        51,
				"n_filters3":        "val1",
			},
			Metric: 2.2962841987609863,
		},
		{
			TrialID: 13,
			Hparams: map[string]interface{}{
				"dropout1":          0.87,
				"dropout2":          0.23,
				"global_batch_size": 64,
				"learning_rate":     0.089,
				"n_filters1":        19,
				"n_filters2":        30,
				"n_filters3":        "val1",
			},
			Metric: 2.2999706268310547,
		},
	}
	nTreesResults, err = createDataFile(data, expConfig, ".")
	assert.NilError(t, err)
	assert.Equal(t, nTreesResults, 10)

	_, err = computeHPImportance(data, expConfig, masterConfig, "growforest", ".")
	assert.Assert(t, err != nil)
}
