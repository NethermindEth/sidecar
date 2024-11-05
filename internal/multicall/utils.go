package multicall

// imagine if golang had a standard library
func mapCollection[A any, B any](coll []A, mapper func(i A, index uint64) B) []B {
	out := make([]B, len(coll))
	for i, item := range coll {
		out[i] = mapper(item, uint64(i))
	}
	return out
}

func filterCollection[A any](coll []A, criteria func(i A) bool) []A {
	out := []A{}
	for _, item := range coll {
		if criteria(item) {
			out = append(out, item)
		}
	}
	return out
}

/*
 * Some RPC providers may limit the amount of calldata you can send in one eth_call, which (for those who have 1000's of validators), means
 * you can't just spam one enormous multicall request.
 *
 * This function checks whether the calldata appended exceeds maxBatchSizeBytes
 */
func chunkCalls(allCalls []ParamMulticall3Call3, maxBatchSizeBytes int) [][]ParamMulticall3Call3 {
	// chunk by the maximum size of calldata, which is 1024 per call.
	results := [][]ParamMulticall3Call3{}
	currentBatchSize := 0
	currentBatch := []ParamMulticall3Call3{}

	for _, call := range allCalls {
		if (currentBatchSize + len(call.CallData)) > maxBatchSizeBytes {
			// we can't fit in this batch, so dump the current batch and start a new one
			results = append(results, currentBatch)
			currentBatchSize = 0
			currentBatch = []ParamMulticall3Call3{}
		}

		currentBatch = append(currentBatch, call)
		currentBatchSize += len(call.CallData)
	}

	// check if we forgot to add the last batch
	if len(currentBatch) > 0 {
		results = append(results, currentBatch)
	}

	return results
}
