package rmevents

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

var defaultManager = newManager()

func Subscribe(topic model.AllocationID) *sproto.AllocationSubscription {
	return defaultManager.subscribe(topic)
}

func Publish(topic model.AllocationID, event sproto.AllocationEvent) {
	defaultManager.publish(topic, event)
}
