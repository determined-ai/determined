package expconf

//go:generate ../gen.sh
// ProfilingConfigV0 configures profiling in the harness.
type ProfilingConfigV0 struct {
	RawEnabled       *bool `json:"enabled"`
	RawBeginOnBatch  *int  `json:"begin_on_batch"`
	RawEndAfterBatch *int  `json:"end_after_batch"`
}
