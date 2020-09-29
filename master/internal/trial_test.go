package internal

import (
	"sort"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

type mockActor struct {
	Messages []interface{}
}

func (a *mockActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case api.WriteMessage:
		a.Messages = append(a.Messages, msg.Message)
		ctx.Respond(api.WriteResponse{})
	default:
	}
	return nil
}

type mockAllocation struct {
}

func (mockAllocation) Summary() resourcemanagers.ContainerSummary {
	return resourcemanagers.ContainerSummary{}
}
func (mockAllocation) Start(ctx *actor.Context, spec tasks.TaskSpec) {}
func (mockAllocation) Kill(ctx *actor.Context)                       {}

func TestRendezvousInfo(t *testing.T) {
	addresses := [][]cproto.Address{
		{
			{
				ContainerPort: 1,
			},
			{
				ContainerPort: MinLocalRendezvousPort + 1,
			},
			{
				ContainerPort: 10,
			},
			{
				ContainerPort: MinLocalRendezvousPort,
			},
			{
				ContainerPort: 100,
			},
		},
		{
			{
				ContainerPort: 200,
			},
			{
				ContainerPort: MinLocalRendezvousPort,
			},
			{
				ContainerPort: 20,
			},
			{
				ContainerPort: MinLocalRendezvousPort + 1,
			},
			{
				ContainerPort: 2,
			},
		},
	}

	system := actor.NewSystem("")

	rp, created := system.ActorOf(
		actor.Addr("resourceManagers"),
		resourcemanagers.NewDeterminedResourceManager(
			resourcemanagers.NewFairShareScheduler(),
			resourcemanagers.WorstFit,
			nil,
			0,
		))
	if !created {
		t.Fatal("unable to create cluster")
	}

	defaultTaskSpec := &tasks.TaskSpec{
		HarnessPath:           "/opt/determined",
		TaskContainerDefaults: model.TaskContainerDefaultsConfig{},
	}

	// This is the minimal trial to receive scheduler.ContainerStarted messages.
	trial := &trial{
		rp:                   rp,
		experiment:           &model.Experiment{},
		task:                 &resourcemanagers.AllocateRequest{},
		allocations:          []resourcemanagers.Allocation{mockAllocation{}, mockAllocation{}},
		experimentState:      model.ActiveState,
		startedContainers:    make(map[cproto.ID]bool),
		terminatedContainers: make(map[cproto.ID]terminatedContainerWithState),
		containers:           make(map[cproto.ID]cproto.Container),
		containerRanks:       make(map[cproto.ID]int),
		containerAddresses:   make(map[cproto.ID][]cproto.Address),
		containerSockets:     make(map[cproto.ID]*actor.Ref),
		taskSpec:             defaultTaskSpec,
	}
	trialRef, created := system.ActorOf(actor.Addr("trial"), trial)
	if !created {
		t.Fatal("unable to create trial")
	}

	// Simulate a stray websocket connecting to the trial.
	strayID := cproto.ID("stray-container-id")
	system.Ask(trialRef, containerConnected{ContainerID: strayID})
	t.Run("Stray sockets are not accepted", func(t *testing.T) {
		_, strayRemains := trial.containerSockets[strayID]
		assert.Assert(t, !strayRemains)
	})

	containers := make([]*cproto.Container, 0)
	mockActors := make(map[*cproto.Container]*mockActor)
	for idx, caddrs := range addresses {
		c := &cproto.Container{
			ID:    cproto.ID(strconv.Itoa(idx)),
			State: cproto.Running,
		}
		mockActors[c] = &mockActor{}
		ref, created := system.ActorOf(actor.Addr(uuid.New().String()), mockActors[c])
		if !created {
			t.Fatal("cannot make socket")
		}

		// Simulate trial containers connecting to the trial actor.
		trial.containerSockets[c.ID] = ref

		// Simulate the scheduling of a container.
		system.Ask(trialRef, sproto.TaskContainerStateChanged{
			Container: *c,
			ContainerStarted: &sproto.TaskContainerStarted{
				Addresses: caddrs,
			},
		}).Get()

		containers = append(containers, c)
	}

	var rmsgs []*rendezvousInfoMessage
	for _, c := range containers {
		for _, msg := range mockActors[c].Messages {
			tmsg, ok := msg.(*trialMessage)
			if !ok {
				continue
			} else if tmsg.RendezvousInfo == nil {
				continue
			}
			rmsgs = append(rmsgs, tmsg.RendezvousInfo)
		}
	}

	if e, f := len(addresses), len(rmsgs); e != f {
		t.Fatalf("expected %d messages but found %d instead", e, f)
	}

	rep := rmsgs[0]

	t.Run("Container addresses are sorted by ContainerPort", func(t *testing.T) {
		for _, c := range rep.Containers {
			var cports []int
			for _, addr := range c.Addresses {
				cports = append(cports, addr.ContainerPort)
			}
			assert.Assert(t, sort.IntsAreSorted(cports), cports)
		}
	})

	t.Run("Rendezvous addrs are sorted", func(t *testing.T) {
		var addrs []int
		for _, addr := range rep.Addrs {
			i, _ := strconv.Atoi(addr)
			addrs = append(addrs, i)
		}
		assert.Assert(t, sort.IntsAreSorted(addrs), addrs)
	})

	t.Run("Rendezvous addrs2 are sorted", func(t *testing.T) {
		var addrs2 []int
		for _, addr := range rep.Addrs2 {
			i, _ := strconv.Atoi(addr)
			addrs2 = append(addrs2, i)
		}
		assert.Assert(t, sort.IntsAreSorted(addrs2), addrs2)
	})

	t.Run("Rendezvous information is the same for all containers", func(t *testing.T) {
		for idx, n := 1, len(rmsgs); idx < n; idx++ {
			// Ignore the rank in comparisons.
			rmsgs[idx].Rank = 0
			assert.DeepEqual(t, rep, rmsgs[idx])
		}
	})
}
