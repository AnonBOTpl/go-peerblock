package filter

import "runtime"

// RecommendedWorkerCount returns the optimal number of pipeline workers.
func RecommendedWorkerCount() int {
	n := runtime.NumCPU()
	if n > 8 {
		return 8
	}
	return n
}
