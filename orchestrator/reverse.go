package orchestrator

func Reverse(jobsSliceOfSlices [][]Job) [][]Job {
	reversedJobs := [][]Job{}
	for _, jobSlice := range jobsSliceOfSlices {
		reversedJobs = append([][]Job{jobSlice}, reversedJobs...)
	}
	return reversedJobs
}
