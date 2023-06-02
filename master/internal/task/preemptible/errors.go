package preemptible

import "fmt"

// ErrPreemptionTimeoutExceeded indicates that an allocation not halt within the expected deadline.
var ErrPreemptionTimeoutExceeded = fmt.Errorf("allocation did not preempt in %s", DefaultTimeout)

// ErrPreemptionDisabled indicates that an alloction is either non-preemptible or not running.
var ErrPreemptionDisabled = fmt.Errorf("allocation is not preemptible")
