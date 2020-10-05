package logs

// Record represents a single record in a batch.
type Record interface{}

// Batch represents a batch of logs.
type Batch interface {
	ForEach(func(Record) error) error
	Size() int
}

// Fetcher fetches returns sequential batches of logs with a limit.
type Fetcher interface {
	Fetch(limit int, unlimited bool) (Batch, error)
}

// OnBatchFn is a callback called on each batch of log entries.
// It returns an error and how many records were processed in the batch.
type OnBatchFn func(Batch) error

// TerminationCheckFn checks whether the batch processing should stop or not.
type TerminationCheckFn func() (bool, error)
