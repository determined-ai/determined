package kubernetesrm

import (
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

type kubernetesResourcePool struct {
	config *config.KubernetesResourceManagerConfig

	reqList           *tasklist.TaskList
	groups            map[*actor.Ref]*tasklist.Group
	addrToContainerID map[*actor.Ref]cproto.ID
	containerIDtoAddr map[string]*actor.Ref
	jobIDtoAddr       map[model.JobID]*actor.Ref
	addrToJobID       map[*actor.Ref]model.JobID
	groupActorToID    map[*actor.Ref]model.JobID
	IDToGroupActor    map[model.JobID]*actor.Ref
	slotsUsedPerGroup map[*tasklist.Group]int

	podsActor *actor.Ref

	queuePositions tasklist.JobSortState
	reschedule     bool
}

func (k *kubernetesResourcePool) Receive(ctx *actor.Context) error {
	reschedule := true
	defer func() {
		// Default to scheduling every 500ms if a message was received, but allow messages
		// that don't affect the cluster to be skipped.
		k.reschedule = k.reschedule || reschedule
	}()

	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, ActionCoolDown, SchedulerTick{})

	case
		tasklist.GroupActorStopped,
		sproto.SetGroupMaxSlots,
		sproto.SetAllocationName,
		sproto.AllocateRequest,
		sproto.ResourcesReleased,
		sproto.UpdatePodStatus,
		sproto.PendingPreemption:
		return k.receiveRequestMsg(ctx)

	case
		sproto.GetJobQ,
		sproto.GetJobQStats,
		sproto.SetGroupWeight,
		sproto.SetGroupPriority,
		sproto.MoveJob,
		sproto.DeleteJob,
		sproto.RecoverJobPosition,
		*apiv1.GetJobQueueStatsRequest:
		return k.receiveJobQueueMsg(ctx)

	case sproto.GetAllocationHandler:
		reschedule = false
		ctx.Respond(k.reqList.TaskHandler(msg.ID))

	case sproto.GetAllocationSummary:
		if resp := k.reqList.TaskSummary(
			msg.ID, k.groups, kubernetesScheduler); resp != nil {
			ctx.Respond(*resp)
		}
		reschedule = false

	case sproto.GetAllocationSummaries:
		reschedule = false
		ctx.Respond(k.reqList.TaskSummaries(k.groups, kubernetesScheduler))

	case SchedulerTick:
		if k.reschedule {
			k.schedulePendingTasks(ctx)
		}
		k.reschedule = false
		reschedule = false
		actors.NotifyAfter(ctx, ActionCoolDown, SchedulerTick{})

	case *apiv1.GetResourcePoolsRequest:
		if summary, err := k.summarizeResourcePool(ctx); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(summary)
		}

	default:
		reschedule = false
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}
