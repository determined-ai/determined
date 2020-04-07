package searcher

func trainAndValidate(requestID RequestID, prevStep, nextStep int) (ops []Operation) {
	for step := prevStep + 1; step <= nextStep; step++ {
		ops = append(ops, NewTrain(requestID, step))
	}
	ops = append(ops, NewValidate(requestID, nextStep))
	return ops
}
