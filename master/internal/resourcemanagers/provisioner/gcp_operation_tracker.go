package provisioner

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/api/compute/v1"

	"github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
)

type (
	trackerTimeout     struct{}
	trackerTick        struct{}
	trackOperationDone struct {
		op *compute.Operation

		doneOp *compute.Operation
		err    error
	}
)

// gcpOperationTracker is an actor that tracks GCP zone operation. Its lifecycle is bound
// with the operation the actor tracks. If the actor is alive, it tracks a GCP operation
// every one second.
type gcpOperationTracker struct {
	config *provconfig.GCPClusterConfig
	client *compute.Service

	op      *compute.Operation
	timeout time.Duration
}

func (t *gcpOperationTracker) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(ctx.Self(), trackerTick{})
		actors.NotifyAfter(ctx, t.timeout, trackerTimeout{})

	case trackerTimeout:
		ctx.Tell(ctx.Self().Parent(), trackOperationDone{
			op:  t.op,
			err: errors.New("tracking GCP operation is timeout"),
		})
		ctx.Self().Stop()

	case trackerTick:
		switch res, err := t.pollOperation(); {
		case err != nil:
			ctx.Log().WithError(err).Error("")
			ctx.Self().Stop()
		case res != nil:
			ctx.Tell(ctx.Self().Parent(), *res)
			ctx.Self().Stop()
		default:
			actors.NotifyAfter(ctx, time.Second, trackerTick{})
		}

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (t *gcpOperationTracker) pollOperation() (*trackOperationDone, error) {
	callCtx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()
	resp, respErr := t.client.ZoneOperations.
		Get(t.config.Project, t.config.Zone, strconv.FormatUint(t.op.Id, 10)).Context(callCtx).Do()
	switch {
	case respErr != nil:
		return &trackOperationDone{
			op: t.op,
			err: errors.Wrapf(
				respErr,
				"GCE cannot track %q operation %q targeting %q",
				t.op.OperationType,
				strconv.FormatUint(t.op.Id, 10),
				t.op.TargetLink,
			),
		}, nil
	case resp.Error != nil:
		// Stop tracking a operation even if it is still running as long as it has error.
		return &trackOperationDone{
			op: t.op,
			err: errors.Errorf(
				"GCE cannot finish %q operation %q targeting %q",
				resp.OperationType,
				strconv.FormatUint(t.op.Id, 10),
				t.op.TargetLink,
			),
			doneOp: resp,
		}, nil
	case resp.Status == "DONE":
		// Stop tracking a operation when it's done and has no errors.
		return &trackOperationDone{
			op:     t.op,
			doneOp: resp,
		}, nil
	case resp.Status == "RUNNING" || resp.Status == "PENDING":
		return nil, nil
	default:
		errOp, _ := json.Marshal(resp)
		return nil, errors.Errorf("unexpected message: %s", errOp)
	}
}

type gcpBatchOperationTracker struct {
	config *provconfig.GCPClusterConfig
	client *compute.Service

	ops         []*compute.Operation
	postProcess func(doneOps []*compute.Operation)
	doneOps     []trackOperationDone
}

func (t *gcpBatchOperationTracker) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		//nolint:lll // There isn't a great way to break this line that makes it more readable.
		batchOperationTimeoutPeriod := time.Duration(len(t.ops)) * time.Duration(t.config.OperationTimeoutPeriod)
		actors.NotifyAfter(ctx, batchOperationTimeoutPeriod, trackerTimeout{})
		t.doneOps = make([]trackOperationDone, 0, len(t.ops))
		for _, op := range t.ops {
			if _, ok := ctx.ActorOf(
				fmt.Sprintf("track-operation-%d", op.Id),
				&gcpOperationTracker{
					config:  t.config,
					client:  t.client,
					op:      op,
					timeout: time.Duration(t.config.OperationTimeoutPeriod),
				},
			); !ok {
				return errors.New("internal error tracking GCP operation")
			}
		}

	case trackOperationDone:
		if msg.err != nil {
			ctx.Log().WithError(msg.err).Error("")
		}
		if msg.doneOp != nil {
			if msg.doneOp.Error != nil {
				for _, err := range msg.doneOp.Error.Errors {
					ctx.Log().Errorf(
						"GCE throws out error (code %s) for operation %q targeting %q: %s",
						err.Code,
						strconv.FormatUint(msg.doneOp.Id, 10),
						msg.doneOp.TargetLink,
						err.Message,
					)
				}
			}
			for _, warning := range msg.doneOp.Warnings {
				ctx.Log().Warnf(
					"GCE throws out warning (code %s) for operation %q targeting %q: %s",
					warning.Code,
					strconv.FormatUint(msg.doneOp.Id, 10),
					msg.doneOp.TargetLink,
					warning.Message,
				)
			}
		}
		t.doneOps = append(t.doneOps, msg)
		if len(t.doneOps) == len(t.ops) {
			ctx.Self().Stop()
		}

	case trackerTimeout:
		ctx.Log().Error("tracking batch GCP operation is timeout")
		ctx.Self().Stop()

	case actor.ChildFailed:
		ctx.Log().WithError(msg.Error).Error("internal error")

	case actor.PostStop:
		successful := make([]*compute.Operation, 0, len(t.doneOps))
		for _, op := range t.doneOps {
			if op.doneOp == nil {
				continue
			}
			successful = append(successful, op.doneOp)
		}
		t.postProcess(successful)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
