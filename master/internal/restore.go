package internal

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

// The current experiment snapshot version. Once this is incremented, older versions should be
// shimmed.
const experimentSnapshotVersion = 0

// Restore works by restoring from distributed consistent snapshots taken through the course
// of an experiment. Snapshots within the system flow from the bottom up, starting with the
// trial workload sequencer, to the trial, and finally to the experiment. Any event that the
// trial or trial workload sequencer processes that would trigger a change to the state of the
// experiment is:
//   1. Propagated atomically, within a single message, to ensure the experiment handles it all
//      or nothing
//   2. With a snapshot affixed to it, to mark that it should trigger a snapshot
// Upon receipt, the experiment handles the event entirely, snapshots its state and saves it
// along with all the snapshots it has received, atomically.
//
// This is technically equivalent to the Chandy-Lamport distributed snapshotting algorithm where
// the trial workload sequencer initiates the snapshots and propagates them along with trialCreated
// and trialCompletedWorkload messages, with the snapshot itself acting as the marker. Each actor we
// care about and know has updated state in the system snapshots it state when it receives this
// marker and propagates it through the system. Even though all trials are part of each distributed
// snapshot, we do not propagate snapshots to trials that didn't initiate the snapshot; that a trial
// sends a snapshot with each state change implies if it did not send the snapshot, it does not have
// a state change [1]. We can also be sure there are no pre-snapshot messages floating in the ether,
// since any message we care about snapshotting is itself a snapshot.
//
// To restore properly, any message the experiment would send as a result of snapshot-able event
// must be replayed on restore. In our case, these are the new operations created by the searcher
// as a result a trialCreated or trialCompletedWorkload message. We restore trials and experiment
// directly from their snapshots and tell the trial all its operations on creation.
//
// [1] This is fine, but _also_ we have to make sure that nothing else alters trial and experiment
// state that doesn't flow from trial to experiment; experiments can never push non-ephemeral state
// updates to trials without special consideration. searcher.Operations are an example of this (and
// the experiment snapshots them and re-sends them).
func (m *Master) restoreExperiment(expModel *model.Experiment) error {
	// Experiments which were trying to stop need to be marked as terminal in the database.
	if terminal, ok := model.StoppingToTerminalStates[expModel.State]; ok {
		if err := m.db.TerminateExperimentInRestart(expModel.ID, terminal); err != nil {
			return errors.Wrapf(err, "terminating experiment %d", expModel.ID)
		}
		expModel.State = terminal
		telemetry.ReportExperimentStateChanged(m.system, m.db, *expModel)
		return nil
	} else if _, ok := model.RunningStates[expModel.State]; !ok {
		return errors.Errorf(
			"cannot restore experiment %d from state %v", expModel.ID, expModel.State,
		)
	}

	log.WithField("experiment", expModel.ID).Info("restoring experiment")
	snapshot, err := m.retrieveExperimentSnapshot(expModel)
	if err != nil {
		return errors.Wrapf(err, "failed to restore experiment %d", expModel.ID)
	}
	e, err := newExperiment(m, expModel)
	if err != nil {
		return errors.Wrapf(err, "failed to create experiment %d from model", expModel.ID)
	}
	if snapshot != nil {
		if err := e.Restore(snapshot); err != nil {
			return errors.Wrap(err, "failed to restore experiment")
		}
		e.restored = true
	}
	log.WithField("experiment", e.ID).Info("restored experiment")

	m.system.ActorOf(actor.Addr("experiments", e.ID), e)
	return nil
}

// restoreTrial takes the a searcher.Create and attempts to restore the trial that would be
// associated with it. On failure, the trial is just reset to the start and errors are logged.
func (e *experiment) restoreTrial(
	ctx *actor.Context, op searcher.Create, ckpt *model.Checkpoint, ops []searcher.Operation,
) (terminal bool) {
	l := ctx.Log().WithField("request-id", op.RequestID)
	l.Info("restoring trial")

	var trialID *int
	var snapshot []byte
	switch trial, err := e.db.TrialByExperimentAndRequestID(e.ID, op.RequestID); {
	case errors.Cause(err) == db.ErrNotFound:
		l.Info("trial was never previously allocated")
	case err != nil:
		// This is the only place we _have_ to error, because if the trial did previously exist
		// and we failed to retrieve it, continuing will result in an invalid state (we'll get a
		// new trial in the trials table with the same (experiment_id, request_id).
		l.WithError(err).Error("failed to retrieve trial, aborting restore")
		return true
	default:
		trialID = &trial.ID
		l = l.WithField("trial-id", trialID)
		if _, terminal = model.TerminalStates[trial.State]; terminal {
			l.Infof("trial was in terminal state in restore: %s", trial.State)
			return true
		} else if _, running := model.RunningStates[trial.State]; !running {
			l.Infof("cannot restore trial in state: %s", trial.State)
			return true
		}
		if snapshot, err = e.retrieveTrialSnapshot(l, op); err != nil {
			l.Warnf("failed to retrieve trial snapshot, restarting fresh: %s", err)
		}
	}

	t := newTrial(e, op, ckpt)
	if trialID != nil {
		t.processID(*trialID)
	}
	if snapshot != nil {
		if err := t.Restore(snapshot); err != nil {
			l.WithError(err).Warn("failed to restore trial, restarting fresh")
			// Just new up the trial again in case restore half-worked.
			t = newTrial(e, op, ckpt)
			if trialID != nil {
				t.processID(*trialID)
			}
		}
	}
	t.replayCreate = trialID != nil && snapshot == nil
	t.processOperations(ops)
	ctx.ActorOf(op.RequestID, t)
	l.Infof("restored trial to the beginning of step %d", t.sequencer.CurStepID)
	return false
}

// retrieveExperimentSnapshot retrieves a snapshot in from database if it exists.
func (m *Master) retrieveExperimentSnapshot(expModel *model.Experiment) ([]byte, error) {
	switch b, err := m.db.ExperimentSnapshot(expModel.ID); {
	case errors.Cause(err) == db.ErrNotFound:
		log.WithField("experiment-id", expModel.ID).Info("no snapshot found")
		return nil, nil
	case err != nil:
		return nil, errors.Wrap(err, "failed to retrieve experiment snapshot")
	default:
		return b, nil
	}
}

func (e *experiment) retrieveTrialSnapshot(
	l *log.Entry, create searcher.Create,
) (snapshot []byte, err error) {
	switch snapshot, err := e.db.TrialSnapshot(e.ID, create.RequestID); {
	case errors.Cause(err) == db.ErrNotFound:
		// This can only happen if the master dies between when the trial saves itself
		// to the database and the trialCreated message is received and handled. If we're here, the
		// easiest fix is to just replay the trial created message.
		l.Info("trial was previously allocated but had no snapshot")
		return nil, nil
	case err != nil:
		return nil, errors.Wrap(err, "failed to retrieve trial snapshot")
	default:
		return snapshot, nil
	}
}

func (e *experiment) snapshotAndSave(ctx *actor.Context, ts trialSnapshot) {
	es, err := e.Snapshot()
	if err != nil {
		e.faultToleranceEnabled = false
		ctx.Log().WithError(err).Errorf("failed to snapshot experiment, fault tolerance is lost")
		return
	}
	err = e.db.SaveSnapshot(e.ID, ts.trialID, ts.requestID, experimentSnapshotVersion, es, ts.snapshot)
	if err != nil {
		e.faultToleranceEnabled = false
		ctx.Log().WithError(err).Errorf("failed to persist experiment snapshot, fault tolerance is lost")
		return
	}
}
