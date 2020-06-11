package searcher

func trainAndValidate(requestID RequestID, prevStep, nextStep, batchesPerStep int) (ops []Operation) {
	for step := prevStep + 1; step <= nextStep; step++ {
		ops = append(ops, NewTrain(requestID, step, batchesPerStep))
	}
	ops = append(ops, NewValidate(requestID, nextStep))
	return ops
}
