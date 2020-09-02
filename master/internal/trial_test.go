package internal

import (
	"sort"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/model"
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

type mockContainer struct {
	id        string
	slots     int
	addresses []scheduler.Address
	mockActor mockActor
	isLeader  bool
}

func (c *mockContainer) ID() scheduler.ContainerID          { return scheduler.ContainerID(c.id) }
func (c *mockContainer) TaskID() scheduler.TaskID           { panic("not implemented") }
func (c *mockContainer) Slots() int                         { return c.slots }
func (c *mockContainer) Addresses() []scheduler.Address     { return c.addresses }
func (c *mockContainer) IsLeader() bool                     { return c.isLeader }
func (c *mockContainer) ExitStatus() agent.ContainerStopped { panic("not implemented") }
func (c *mockContainer) Tell(message actor.Message)         {}

func TestRendezvousInfo(t *testing.T) {
	addresses := [][]scheduler.Address{
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
		actor.Addr("resourceProviders"),
		scheduler.NewDefaultRP(
			uuid.New().String(),
			scheduler.NewFairShareScheduler(),
			scheduler.WorstFit,
			"/opt/determined",
			model.TaskContainerDefaultsConfig{},
			nil,
			0,
			nil,
		))
	if !created {
		t.Fatal("unable to create cluster")
	}

	// This is the minimal trial to receive scheduler.ContainerStarted messages.
	trial := &trial{
		rp:            rp,
		experiment:    &model.Experiment{},
		containers:    make(map[scheduler.ContainerID]scheduler.Container),
		sockets:       make(map[scheduler.ContainerID]*actor.Ref),
		task:          &scheduler.Task{},
		numContainers: len(addresses),
	}
	trialRef, created := system.ActorOf(actor.Addr("trial"), trial)
	if !created {
		t.Fatal("unable to create trial")
	}

	// Simulate a stray websocket connecting to the trial.
	var stray mockActor
	strayRef, created := system.ActorOf(actor.Addr(uuid.New().String()), &stray)
	if !created {
		t.Fatal("unable to create cluster")
	}
	strayID := scheduler.ContainerID("stray-container-id")
	trial.sockets[strayID] = strayRef

	var containers []*mockContainer
	for idx, caddrs := range addresses {
		c := &mockContainer{
			id:        strconv.Itoa(idx),
			slots:     1,
			addresses: caddrs,
			isLeader:  idx == 0,
		}
		ref, created := system.ActorOf(actor.Addr(uuid.New().String()), &c.mockActor)
		if !created {
			t.Fatal("cannot make socket")
		}

		// Simulate trial containers connecting to the trial actor.
		trial.sockets[c.ID()] = ref

		// Simulate the scheduling of a container.
		system.Ask(trialRef, scheduler.ContainerStarted{
			Container: c,
		}).Get()

		containers = append(containers, c)
	}

	t.Run("Stray sockets are dropped", func(t *testing.T) {
		_, strayRemains := trial.sockets[strayID]
		assert.Assert(t, !strayRemains)
	})

	var rmsgs []*rendezvousInfoMessage
	for _, c := range containers {
		for _, msg := range c.mockActor.Messages {
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
