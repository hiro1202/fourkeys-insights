package jobs

// ProgressCallback is called during sync to report progress.
type ProgressCallback func(fetched, total int, currentRepo string)
