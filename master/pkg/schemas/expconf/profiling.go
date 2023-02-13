package expconf

// ProfilingConfigV0 configures profiling in the harness.
//
//go:generate ../gen.sh
type ProfilingConfigV0 struct {
	RawEnabled       *bool `json:"enabled"`
	RawBeginOnBatch  *int  `json:"begin_on_batch"`
	RawEndAfterBatch *int  `json:"end_after_batch"`
	RawSyncTimings   *bool `json:"sync_timings"`
}
