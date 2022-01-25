package resourcemanagers

import (
	"github.com/determined-ai/determined/master/internal/db"
)

// type SnapShot struct {
// 	pools map[string]*SchedluingSnapShot
// }

type SnapShotID = string // resource pool name. maybe a combo of rp and rm type?

// used at RM.setup()
func retrieveSnapshot(db *db.PgDB, id SnapShotID) (*persistedState, error) {
	// go to db and fetch rm state SnapShot
	// mold it into SchedluingSnapShot

	return nil, nil
}

// used at RP level in agent. and rm level in k8
func saveSnapshot(db *db.PgDB, snapshot *persistedState, id SnapShotID) error {
	// convert it into a snapshot

	// go to db and save rm state
	return nil
}
