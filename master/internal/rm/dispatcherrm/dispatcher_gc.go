package dispatcherrm

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
)

const (
	// Delay before gcOrphanedDispatches starts its work.
	gcDelay = 15 * time.Minute
)

// gcOrphanedDispatches is a background method to do
// a single garbage collection of orhaned dispatches.  Such
// orphaned dispatches should be rare in normal circumstances, but
// due to defects, communication errors with the launcher, or
// changes to the job_storage_root configuraition we may have unused
// dispatches stored by the launcher.   Excessive unused dispatches
// take up disk space, and increases search time for certain launcher
// operations.  This method waits for gcDelay time beforing doing its
// work to reduce impact on startup time.   It does a terminate of
// on all orphaned running dispatches (orphaned PENDING/RUNNING need an
// explicit termination to get to a terminated state -- in case there is no
// job ID to check).   It then deletes all orphaned terminated dispatches.
// Once gc completes, the  routine terminates.
func gcOrphanedDispatches(
	ctx context.Context,
	log *logrus.Entry,
	cl *launcherAPIClient,
) {
	time.Sleep(gcDelay)
	gcTerminateRunningOrphans(ctx, log, cl)
	gcTerminatedOrphans(ctx, log, cl)
}

// refreshRunningOrphans requests the launcher to terminate and update the state of
// all dispatches not currently in-use by Determined as identified by the entries in the db.
// This enables safe gc of terminated jobs using the gcTerminated method.
func gcTerminateRunningOrphans(
	ctx context.Context,
	log *logrus.Entry,
	cl *launcherAPIClient,
) {
	data, _, err := cl.RunningApi.
		ListAllRunning(cl.withAuth(ctx)).
		EventLimit(0).
		Execute() //nolint:bodyclose
	if err != nil {
		log.WithError(err).Warnf("Unable to list running dispatches. skipping status refresh for gc")
		return
	}

	dispatches := data["data"]
	log.Infof("gc found %d running dispatches", len(dispatches))
	for _, v := range dispatches {
		dispatchID := v.GetDispatchId()
		_, err := db.DispatchByID(ctx, dispatchID)
		if err == nil {
			log.Debugf("Dispatch still referenced %s %s", dispatchID, v.GetState())
			continue
		}

		owner := *v.GetLaunchedCapsuleReference().Owner
		log.Infof("Terminate DispatchID %s %s %s", dispatchID, owner, v.GetState())
		_, _, err = cl.terminateDispatch(owner, dispatchID) //nolint:bodyclose
		if err != nil {
			log.WithError(err).Warnf("Unable to terminate DispatchID %s", dispatchID)
			continue
		}

		dispatch, _, err := cl.MonitoringApi.
			GetEnvironmentStatus(cl.withAuth(ctx), owner, dispatchID).
			Refresh(true).
			Execute() //nolint:bodyclose
		if err != nil {
			log.WithError(err).Warnf("Unable to refresh status of DispatchID %s", dispatchID)
		} else {
			log.Infof("Refreshed  DispatchID %s %s %s", dispatchID, owner, dispatch.GetState())
		}
	}
}

// gcTerminatedOrphans processes all terminated dispatches and deletes any that are
// not referenced in the determined db.
func gcTerminatedOrphans(
	ctx context.Context,
	log *logrus.Entry,
	cl *launcherAPIClient,
) {
	data, _, err := cl.TerminatedApi.
		ListAllTerminated(cl.withAuth(ctx)).
		EventLimit(0).
		Execute() //nolint:bodyclose
	if err != nil {
		log.WithError(err).Warnf("Unable to list terminated dispatches. Skipping dispatch gc")
		return
	}

	dispatches := data["data"]
	log.Infof("gc found %d terminated dispatches", len(dispatches))

	for _, v := range dispatches {
		dispatchID := v.GetDispatchId()
		_, err := db.DispatchByID(ctx, dispatchID)
		if err == nil {
			log.Debugf("gc DispatchID still referenced %s", dispatchID)
			continue
		}

		owner := *v.GetLaunchedCapsuleReference().Owner
		log.Infof("gc DispatchID %s %s %s", dispatchID, owner, v.GetState())
		_, err = cl.deleteDispatch(owner, dispatchID) //nolint:bodyclose
		if err != nil {
			log.WithError(err).Errorf("Unable to gc DispatchID %s %s", dispatchID, owner)
		}
	}
}
