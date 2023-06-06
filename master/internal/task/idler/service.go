package idler

import (
	"time"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/syncx/mapx"
)

var idlers = mapx.New[string, *Watcher]()

// Register a idler to default service. The action is called at most once when the idle timeout is
// exceeded. The action can trigger until Unregister is called.
// ID must be a globally unique identifier for the idler.
func Register(id string, cfg *sproto.IdleTimeoutConfig, action func(error)) {
	idlers.Store(id, New(cfg, action))
}

// Unregister removes a idler from the service.
// ID must be a globally unique identifier for the idler.
func Unregister(id string) {
	iw, ok := idlers.Delete(id)
	if !ok {
		return
	}
	iw.Close()
}

// RecordActivity records activity for a idler.
// ID must be a globally unique identifier for the idler.
func RecordActivity(id string) {
	iw, ok := idlers.Load(id)
	if !ok {
		return
	}
	iw.RecordActivity(time.Now())
}
