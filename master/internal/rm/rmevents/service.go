package rmevents

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

var defaultManager = newManager()

// Subscribe to events for the given allocation. Returns a subscription with a channel to listen
// to events. When finished listening, callers should call `AllocationSubscription.Close()`.
func Subscribe(topic model.AllocationID) *sproto.ResourcesSubscription {
	return defaultManager.subscribe(topic)
}

// Publish an event for the provided allocation.
func Publish(topic model.AllocationID, event sproto.ResourcesEvent) {
	defaultManager.publish(topic, event)
}
