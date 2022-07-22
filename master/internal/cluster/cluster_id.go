package cluster

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
)

type clusterID struct {
	bun.BaseModel `bun:"table:cluster_id"`

	ClusterID        string    `bun:"cluster_id,notnull"`
	ClusterHeartbeat time.Time `bun:"cluster_heartbeat,notnull"`
}

var (
	theLastBootMutex            sync.Mutex
	theLastBootClusterHeartbeat *time.Time
)

// InitTheLastBootClusterHeartbeat preserves the last boot heartbeat for applications that need
// it after the master has been running for some time (e.g. open allocation reattachment).
func InitTheLastBootClusterHeartbeat() {
	theLastBootMutex.Lock()
	defer theLastBootMutex.Unlock()

	if theLastBootClusterHeartbeat != nil {
		log.Warn("detected re-initialization of the last boot cluster heartbeat ts")
	}

	clusterRecord := new(clusterID)
	err := db.Bun().NewSelect().Model(clusterRecord).Scan(context.TODO())
	if err != nil {
		log.WithError(err).Warn("failed to init the last boot cluster heartbeat")
		return
	}

	theLastBootClusterHeartbeat = &clusterRecord.ClusterHeartbeat
}

// TheLastBootClusterHeartbeat returns the last known heartbeat time from the previous master boot.
func TheLastBootClusterHeartbeat() *time.Time {
	return theLastBootClusterHeartbeat
}
