package container

import "github.com/determined-ai/determined/master/pkg/aproto"

// Event is the union of all events emitted by a container. When used, only one should be set.
type Event struct {
	StateChange *aproto.ContainerStateChanged
	Log         *aproto.ContainerLog
	StatsRecord *aproto.ContainerStatsRecord
}
