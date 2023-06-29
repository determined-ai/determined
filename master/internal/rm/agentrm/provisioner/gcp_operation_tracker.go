package provisioner

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/compute/v1"

	"github.com/determined-ai/determined/master/internal/config/provconfig"
)

type (
	trackOperationDone struct {
		op *compute.Operation

		doneOp *compute.Operation
		err    error
	}
)

type doneCallback func(
	op *compute.Operation,
	doneOp *compute.Operation,
	err error,
)

// gcpOperationTracker is an actor that tracks GCP zone operation. Its lifecycle is bound
// with the operation the actor tracks. If the actor is alive, it tracks a GCP operation
// every one second.
type gcpOperationTracker struct {
	config *provconfig.GCPClusterConfig
	client *compute.Service
	op     *compute.Operation
}

func (t *gcpOperationTracker) pollOperation(ctx context.Context, done doneCallback) error {
	for {
		select {
		case <-ctx.Done():
			done(t.op, nil, errors.New("tracking GCP operation timeout"))
			return nil
		default:
		}

		resp, respErr := t.client.ZoneOperations.
			Get(t.config.Project, t.config.Zone, strconv.FormatUint(t.op.Id, 10)).Context(ctx).Do()

		switch {
		case respErr != nil:
			err := errors.Wrapf(
				respErr,
				"GCE cannot track %q operation %q targeting %q",
				t.op.OperationType,
				strconv.FormatUint(t.op.Id, 10),
				t.op.TargetLink,
			)
			done(t.op, nil, err)
			return nil
		case resp.Error != nil:
			// Stop tracking a operation even if it is still running as long as it has error.
			err := errors.Errorf(
				"GCE cannot finish %q operation %q targeting %q",
				resp.OperationType,
				strconv.FormatUint(t.op.Id, 10),
				t.op.TargetLink,
			)
			done(t.op, resp, err)
			return nil
		case resp.Status == "DONE":
			// Stop tracking a operation when it's done and has no errors.
			done(t.op, resp, nil)
			return nil
		case resp.Status == "RUNNING" || resp.Status == "PENDING":
			// Do nothing, keep tracking the operation.
		default:
			errOp, _ := json.Marshal(resp)
			return errors.Errorf("unexpected message: %s", errOp)
		}
		// Slow down the polling rate.
		time.Sleep(time.Second)
	}
}

type gcpBatchOperationTracker struct {
	mu sync.Mutex

	config *provconfig.GCPClusterConfig
	client *compute.Service

	ops     []*compute.Operation
	doneOps []trackOperationDone

	syslog *logrus.Entry
}

func newBatchOperationTracker(
	config *provconfig.GCPClusterConfig,
	client *compute.Service,
	ops []*compute.Operation,
) *gcpBatchOperationTracker {
	return &gcpBatchOperationTracker{
		config:  config,
		client:  client,
		ops:     ops,
		doneOps: make([]trackOperationDone, 0, len(ops)),
		syslog:  logrus.WithField("component", "gcp-batch-operation-tracker"),
	}
}

func (t *gcpBatchOperationTracker) start(postProcess func([]*compute.Operation)) {
	timeout := time.Duration(t.config.OperationTimeoutPeriod)
	groupCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	g := new(errgroup.Group)
	t.doneOps = make([]trackOperationDone, 0, len(t.ops))
	for _, op := range t.ops {
		o := &gcpOperationTracker{t.config, t.client, op}
		g.Go(func() error {
			return o.pollOperation(groupCtx, t.trackOperationDone)
		})
	}
	err := g.Wait()
	if err != nil {
		t.syslog.WithError(err).Error("tracking batch GCP operation failed")
	}
	successful := make([]*compute.Operation, 0, len(t.doneOps))
	for _, op := range t.doneOps {
		if op.doneOp == nil {
			continue
		}
		successful = append(successful, op.doneOp)
	}
	postProcess(successful)
}

func (t *gcpBatchOperationTracker) trackOperationDone(
	op *compute.Operation,
	doneOp *compute.Operation,
	err error,
) {
	t.logErrors(doneOp, err)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.doneOps = append(t.doneOps, trackOperationDone{op, doneOp, err})
}

func (t *gcpBatchOperationTracker) logErrors(doneOp *compute.Operation, err error) {
	if err != nil {
		t.syslog.WithError(err).Error("")
	}
	if doneOp != nil {
		if doneOp.Error != nil {
			for _, err := range doneOp.Error.Errors {
				t.syslog.Errorf(
					"GCE throws out error (code %s) for operation %q targeting %q: %s",
					err.Code,
					strconv.FormatUint(doneOp.Id, 10),
					doneOp.TargetLink,
					err.Message,
				)
			}
		}
		for _, warning := range doneOp.Warnings {
			t.syslog.Warnf(
				"GCE throws out warning (code %s) for operation %q targeting %q: %s",
				warning.Code,
				strconv.FormatUint(doneOp.Id, 10),
				doneOp.TargetLink,
				warning.Message,
			)
		}
	}
}
