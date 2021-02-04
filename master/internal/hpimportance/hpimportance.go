package hpimportance

import (
	"fmt"
	"os"
	"os/exec"
	"log"
	"github.com/determined-ai/determined/master/pkg/model"
	"encoding/csv"
	"io"
	"strconv"
	"sort"
)

const (maxDiffCompBatch=4
	   minNumberTrials=50
	   idealBatchDiff=2
	   nReplications=2)

func createDataFile(data map[int][]model.HPImportanceTrialData, experimentConfig *model.ExperimentConfig, workingDir string) (int){
	f, _ := os.Create(workingDir + "/data.arff")

	// create top of file based on exp config
	f.WriteString("@relation data\n\n")
	f.WriteString("@attribute metric numeric\n")
	var hpsOrder []string // HPs must be in the same order for the arff file
	hps := experimentConfig.Hyperparameters

	for key, element := range hps{

		if element.ConstHyperparameter != nil {
			continue;

		} else if element.CategoricalHyperparameter != nil {
			hpsOrder = append(hpsOrder, key)

			st := "@attribute " + key + " {" 
			f.WriteString(st)

			vals := element.CategoricalHyperparameter.Vals
			for _, cat_val := range vals{
				s := fmt.Sprintf("%v", cat_val)
				f.WriteString(s + ",")
			}
			f.WriteString("}\n")

		} else {
			hpsOrder = append(hpsOrder, key)
			st := "@attribute " + key + " numeric\n"
			f.WriteString(st)
		} 
	}
	st := "@attribute numBatches numeric\n"
	f.WriteString(st)

	// Now we have to add the data to the file
	f.WriteString("\n\n@data\n")

	// First, get batch ids unless better way to get them in go?
	var batches []int
	for k := range data {
		batches = append(batches, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(batches)))

	totalNumTrials := 0
	maxNumBatches := batches[0]
	for _, batchId := range batches{
		if batchId <= maxNumBatches / maxDiffCompBatch{
			break;
		}
		if totalNumTrials > minNumberTrials && batchId <= maxNumBatches / idealBatchDiff{
			break;
		}
		batchIdStr := fmt.Sprintf("%v", batchId)
		for _, trial := range data[batchId]{
			st := ""
			metric := fmt.Sprintf("%v", trial.Metric)
			st = st + metric
			hparamsVals := trial.Hparams
			for _, hp :=  range hpsOrder{
				s := fmt.Sprintf("%v", hparamsVals[hp])
				st = st + "," + s
			}

			st = st + ","+batchIdStr +"\n"
			f.WriteString(st)

			totalNumTrials = totalNumTrials + 1
		}

	}

	f.Close()
	return totalNumTrials
}

func computeHPImportance(data map[int][]model.HPImportanceTrialData, experimentConfig *model.ExperimentConfig, masterConfig HPImportanceConfig, growforest string, workingDir string) (map[string]float64) {


	totalNumTrials := createDataFile(data, experimentConfig, workingDir)

	nCores := strconv.FormatInt(int64(masterConfig.CoresPerWorker), 10)
	maxNumTrees := int64(masterConfig.MaxTrees)
	
	// random may be smaller because only 50 trials are ran
	// where I'm not gonna calculate the random forest
	// TODO: Determine best way to handle small amount of trials
	// if totalNumTrials < minNumberTrials{
	// 	return errors.New("Not enough trials for HP importance.")
	// }

	// For version one, we do half the number of trials up to 300
	// This may be overdoing and running too long for minimal performance inc.
	// The paper uses 2 values replications and ntrees while the library,
	// only uses --ace which is replications.
	// TODO: use the bar chart from hp viz to improve hp importance
	numTrees := int64( totalNumTrials / nReplications)

	if numTrees > maxNumTrees {
		numTrees = maxNumTrees
	}
	strNumTrees := strconv.FormatInt(numTrees, 10)

	// Call Random Forest
	_, err := exec.Command(growforest , "-train", workingDir + "/data.arff", "-target=metric", "-nCores", nCores,"-ace", strNumTrees, "-importance", workingDir + "/importance.txt","-rfpred", workingDir+"rface.sf").Output()
	if err != nil {
        log.Fatal(err)
	}

	// Read and Return Output
	file, err := os.Open(workingDir + "/importance.txt")

	output := make(map[string]float64)
	r := csv.NewReader(file)
	r.Comma = '\t'
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if record[1] == "metric"{
			continue
		}
		output[record[1]], _ = strconv.ParseFloat(record[2], 32) // 32 or 64 bit?
	}
	return output //set up to return more then one variable in worker
}