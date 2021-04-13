package expconf

//go:generate ../gen.sh
// ProfilingConfigV0 configures profiling in the harness.
type ProfilingConfigV0 struct {
	Enabled      *bool `json:"enabled"`
	BeginOnBatch *int  `json:"begin_on_batch"`
	EndOnBatch   *int  `json:"end_on_batch"`
}
