/**
This file computes the HP importance for the HP visualizations.
It uses the CloudForest utility (github.com/ryanbressler/CloudForest).

The core steps are create the data file, run growforest which outputs
the importance, read and return the values.
**/

package hpimportance

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	maxDiffCompBatch = 4
	minNumberTrials  = 50
	idealBatchDiff   = 2
	nReplications    = 2

	arffFile       = "data.arff"
	importanceFile = "importance.txt"
)

// The data needs to be put into an arff format.
// Arff files require variable declaration at the top,
// where order does matter. We need to keep define the Hps and
// keep track of the order so the data columns will match.
func createDataFile(data map[int][]model.HPImportanceTrialData,
	experimentConfig *model.ExperimentConfig, dataFile string) (int, error) {
	f, err := os.Create(dataFile)
	if err != nil {
		return 0, err
	}
	defer func() {
		err = f.Close()
		if err != nil {
			log.WithError(err).Error("failed to close arff file")
		}
	}()

	arff := bufio.NewWriter(f)

	// create top of file based on exp config
	_, err = arff.WriteString("@relation data\n\n@attribute metric numeric\n")
	if err != nil {
		return 0, err
	}
	var hpsOrder []string // HPs must be in the same order for the arff file
	hps := experimentConfig.Hyperparameters

	for key, element := range hps {
		var st string
		switch {
		case element.ConstHyperparameter != nil:
			continue
		case element.CategoricalHyperparameter != nil:
			hpsOrder = append(hpsOrder, key)

			var values string
			for _, catVal := range element.CategoricalHyperparameter.Vals {
				values += fmt.Sprintf("%v,", catVal)
			}
			st = fmt.Sprintf("@attribute %s {%s}\n", key, values)
		default:
			hpsOrder = append(hpsOrder, key)
			st = fmt.Sprintf("@attribute %s numeric\n", key)
		}
		_, err = arff.WriteString(st)
		if err != nil {
			return 0, err
		}
	}
	_, err = arff.WriteString("@attribute numBatches numeric\n\n")
	if err != nil {
		return 0, err
	}

	// Now we have to add the data to the file
	_, err = arff.WriteString("\n@data\n")
	if err != nil {
		return 0, err
	}

	// First, get batch ids unless better way to get them in go?
	var batches []int
	for k := range data {
		batches = append(batches, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(batches)))

	totalNumTrials := 0
	maxNumBatches := batches[0]
	for _, batchID := range batches {
		if batchID < maxNumBatches/maxDiffCompBatch {
			break
		}
		if totalNumTrials > minNumberTrials && batchID <= maxNumBatches/idealBatchDiff {
			break
		}
		batchIDStr := fmt.Sprintf("%v", batchID)
		for _, trial := range data[batchID] {
			var st string
			st += fmt.Sprintf("%v", trial.Metric)
			hparamsVals := trial.Hparams
			for _, hp := range hpsOrder {
				st += fmt.Sprintf(",%v", hparamsVals[hp])
			}
			st += fmt.Sprintf(",%s\n", batchIDStr)
			_, err = arff.WriteString(st)
			if err != nil {
				return 0, err
			}

			totalNumTrials++
		}
	}
	err = arff.Flush()
	if err != nil {
		return 0, err
	}
	return totalNumTrials, nil
}

// Read the data from the output importance file and return as a json.
func parseImportanceOutput(filename string) (map[string]float64, error) {
	// The importance file is the target, hpimportance, p-value, mean difference. We will use the
	// p-value and ignore the target to target and batch features.

	// #nosec G304 // Ignore security warning because none of this is user-provided input
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open HP importance file: %w", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.WithError(err).Error("failed to close HP importance file")
		}
	}()

	hpi := make(map[string]float64)
	r := csv.NewReader(file)
	r.Comma = '\t'
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read HP importance file: %w", err)
		}

		if record[1] == "metric" || record[1] == "numBatches" {
			continue
		}
		hpi[record[1]], err = strconv.ParseFloat(record[2], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse HP importance value: %w", err)
		}
	}
	return hpi, nil
}

// For the implementation, since we need to account for adaptive search
// but we don't want to compare trials that have trained for 10,000 vs 100 batches
// therefore, we continue to add trials till one of 2 conditions are met.
// 1. There are at least 50 trials to train with and the ideal number of trial difference is met.
// 2. The difference between the most(aka max) trained trial and the lowest batch are within a
// defined range (maxDiffCompBatches).
func computeHPImportance(data map[int][]model.HPImportanceTrialData,
	experimentConfig *model.ExperimentConfig, masterConfig HPImportanceConfig,
	growforest string, workingDir string) (map[string]float64, error) {
	growforestInput := path.Join(workingDir, arffFile)
	growforestOutput := path.Join(workingDir, importanceFile)

	totalNumTrials, err := createDataFile(data, experimentConfig, growforestInput)
	if err != nil {
		return nil, fmt.Errorf("error writing ARFF file: %w", err)
	}

	nCores := strconv.FormatInt(int64(masterConfig.CoresPerWorker), 10)
	maxNumTrees := int64(masterConfig.MaxTrees)

	// random may be smaller because only 50 trials are ran
	// where I'm not gonna calculate the random forest
	// TODO: Determine best way to handle small amount of trials
	if totalNumTrials < minNumberTrials {
		return nil, fmt.Errorf("not enough trials for HP importance: %d", totalNumTrials)
	}

	// For version one, we do half the number of trials up to 300
	// This may be overdoing and running too long for minimal performance inc.
	// The paper uses 2 values replications and ntrees while the library,
	// only uses --ace which is replications.
	// TODO: use the bar chart from hp viz to improve hp importance
	numTrees := int64(totalNumTrials / nReplications)

	if numTrees > maxNumTrees {
		numTrees = maxNumTrees
	}
	strNumTrees := strconv.FormatInt(numTrees, 10)

	// Call Random Forest
	// Ignore security warning because none of this is user-provided input
	// #nosec G204
	output, err := exec.Command(growforest,
		// the data file created above
		"-train", growforestInput,
		// the y value used to predict. We use a generic metric so we don't have to keep track
		// of current metric since name doesn't matter
		"-target", "metric",
		// number of CPUs to use
		"-nCores", nCores,
		// number of replications/permutations
		"-ace", strNumTrees,
		// output file
		"-importance", growforestOutput,
		// file name to output predictor forest in sf format.
		"-rfpred", path.Join(workingDir, "rface.sf"),
	).CombinedOutput()
	if err != nil {
		log.Error("growforest failed:\n " + string(output))
		return nil, fmt.Errorf("random forest failed: %w", err)
	}

	hpi, err := parseImportanceOutput(growforestOutput)
	return hpi, err
}
