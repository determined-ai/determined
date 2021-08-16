package task

import (
	"fmt"
	"sort"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	apiutils "github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

const (
	// MinLocalRendezvousPort is the smallest port to use (from the container's point of view;
	// it will be mapped to some arbitrary port on the host) for communication across containers.
	MinLocalRendezvousPort = 1734

	// MaxLocalRendezvousPort is the largest port to use for communication across containers.
	// Each distributed trial can take up to 2 host based ports and we assume a maximum.
	// of 16 slot per agent. MaxLocalRendezvousPort = MinLocalRendezvousPort + 2*16 - 1.
	MaxLocalRendezvousPort = MinLocalRendezvousPort + 2*16 - 1
)

// RendezvousTimeoutDuration is the default timeout for rendezvous.
var RendezvousTimeoutDuration = 10 * time.Minute

type (
	// WatchRendezvousInfo begins watching for rendezvous info.
	// When all the containers are ready, the trial will send all the
	// peer addresses on the channel in the response.
	WatchRendezvousInfo struct {
		AllocationID model.AllocationID
		ContainerID  cproto.ID
	}
	// RendezvousInfoOrError contains either rendezvous info or an error from failing
	// to materialize it.
	RendezvousInfoOrError struct {
		Info *trialv1.RendezvousInfo
		Err  error
	}
	// RendezvousWatcher contains a channel which can be polled for rendezvous info.
	RendezvousWatcher struct {
		C <-chan RendezvousInfoOrError
	}
	// UnwatchRendezvousInfo removes the watcher for the given container.
	UnwatchRendezvousInfo struct{ ID cproto.ID }

	// RendezvousTimeout tracks the timeout of the allocation reservations rendezvousing.
	// It is possible that it takes very long for all containers to be connected after the first
	// container is connected. This might happen when the k8s cluster waits for new instances
	// to spin up, which might not happen at all. At the same time, taking up part of all
	// the resources and waiting is wasteful. So we need to detect this situation.
	RendezvousTimeout struct{ AllocationID model.AllocationID }

	// Rendezvous encapsulates the rendezvous state of a trial.
	Rendezvous struct {
		allocationID      model.AllocationID
		watchers          map[cproto.ID]chan<- RendezvousInfoOrError
		ranks             map[cproto.ID]int
		addresses         map[cproto.ID][]cproto.Address
		lastWatchTime     time.Time
		allReadySucceeded bool
	}
)

// NewRendezvous returns a new rendezvous component.
func NewRendezvous(allocationID model.AllocationID, ranks map[cproto.ID]int) Rendezvous {
	return Rendezvous{
		allocationID: allocationID,
		ranks:        ranks,
		addresses:    map[cproto.ID][]cproto.Address{},
		watchers:     map[cproto.ID]chan<- RendezvousInfoOrError{},
	}
}

// Receive implements actor.Receive.
func (r *Rendezvous) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case WatchRendezvousInfo:
		if w, err := r.watch(msg.AllocationID, msg.ContainerID); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(w)
		}
	case UnwatchRendezvousInfo:
		r.unwatch(msg.ID)
	case RendezvousTimeout:
		if err := r.checkTimeout(msg.AllocationID); err != nil {
			return err
		}
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func ranksFromReservations(allocations []sproto.Reservation) map[cproto.ID]int {
	ranks := map[cproto.ID]int{}
	for rank, a := range allocations {
		ranks[a.Summary().ID] = rank
	}
	return ranks
}

func (r *Rendezvous) watch(
	allocationID model.AllocationID, id cproto.ID,
) (RendezvousWatcher, error) {
	if r.allocationID != allocationID {
		err := ErrStaleAllocation{Received: allocationID, Actual: r.allocationID}
		return RendezvousWatcher{}, apiutils.AsValidationError(err.Error())
	} else if _, ok := r.ranks[id]; !ok {
		err := ErrStaleContainer{ID: id}
		return RendezvousWatcher{}, apiutils.AsValidationError(err.Error())
	} else if _, ok := r.watchers[id]; ok {
		return RendezvousWatcher{}, apiutils.AsValidationError(
			"rendezvous request from already connected container: %s", id,
		)
	}

	// Channel is size 1 since rendezvous info will only ever be sent once.
	w := make(chan RendezvousInfoOrError, 1)
	r.watchers[id] = w
	r.lastWatchTime = time.Now()
	if r.ready() {
		r.push()
	}
	return RendezvousWatcher{C: w}, nil
}

func (r *Rendezvous) unwatch(id cproto.ID) {
	if r == nil {
		return
	}
	delete(r.watchers, id)
}

func (r *Rendezvous) containerStarted(id cproto.ID, addresses []cproto.Address) {
	r.addresses[id] = addresses
	if r.ready() {
		r.push()
	}
}

func (r *Rendezvous) containerTerminated(id cproto.ID) {
	delete(r.addresses, id)
}

func (r Rendezvous) rank(id cproto.ID) int {
	return r.ranks[id]
}

// ready returns true if and only if all the containers are reported to be started with the
// ContainerStarted message and their sockets to be connected with the containerConnected
// message. The two messages are not guaranteed to come in-order. During each run of the
// trial, once all the containers are ready this function will return true afterward because this
// function is used in deciding if the trial should be forcibly killed when terminating.
func (r *Rendezvous) ready() bool {
	// If a trial has passed allReady it can never return to a state of not ready until the
	// current containers are all taskTerminated.
	if r.allReadySucceeded {
		return true
	}

	allAddressesArrived := len(r.addresses) == len(r.ranks)
	allWaiting := len(r.watchers) == len(r.ranks)

	r.allReadySucceeded = allAddressesArrived && allWaiting
	return r.allReadySucceeded
}

// push gathers up the external addresses for the exposed ports and sends them to all the
// containers in the trial.
func (r Rendezvous) push() bool {
	if !r.ready() {
		return false
	}
	caddrs, raddrs, err := r.info()
	for _, caddr := range caddrs {
		w := r.watchers[caddr.id]
		w <- RendezvousInfoOrError{
			Info: &trialv1.RendezvousInfo{
				Addresses: raddrs,
				Rank:      int32(r.ranks[caddr.id]),
			},
			Err: err,
		}
		close(w)
		delete(r.watchers, caddr.id)
	}
	return true
}

// checkTimeout checks if the task should timeout waiting for rendezvous.
func (r *Rendezvous) checkTimeout(allocationID model.AllocationID) error {
	if r == nil {
		return nil
	}

	exceededTimeout := time.Now().After(r.lastWatchTime.Add(RendezvousTimeoutDuration))
	if r.allocationID == allocationID && exceededTimeout {
		return ErrTimeoutExceeded{
			Message: "some containers are taking a long time to " +
				"connect to master; when running on kubernetes this may happen " +
				"because only some of the pods have been scheduled; it is possible " +
				"that some pods will never be scheduled without adding compute " +
				"resources or pausing / killing other experiments in the cluster",
		}
	}
	return nil
}

func (r *Rendezvous) close() {
	if r == nil {
		return
	}

	for cID, w := range r.watchers {
		w <- RendezvousInfoOrError{Err: errors.New("task taskTerminated")}
		close(w)
		delete(r.watchers, cID)
	}
}

type cAddress struct {
	id        cproto.ID
	addresses []cproto.Address
	ordinal   int
}

func (r *Rendezvous) info() ([]cAddress, []string, error) {
	var caddrs []cAddress
	for id, rank := range r.ranks {
		caddr := cAddress{
			id:        id,
			addresses: r.addresses[id],
			ordinal:   rank,
		}
		caddrs = append(caddrs, caddr)

		sort.Slice(caddr.addresses, func(i, j int) bool {
			a := caddr.addresses[i]
			b := caddr.addresses[j]

			return a.ContainerPort < b.ContainerPort
		})
	}

	sort.Slice(caddrs, func(i, j int) bool {
		a := caddrs[i]
		b := caddrs[j]
		switch {
		case a.ordinal == 0 && b.ordinal != 0:
			return true
		case a.ordinal != 0 && b.ordinal == 0:
			return false
		default:
			return a.id < b.id
		}
	})

	var raddrs []string
	var err *multierror.Error
	for _, caddr := range caddrs {
		var addrs []cproto.Address
		for _, addr := range caddr.addresses {
			if MinLocalRendezvousPort <= addr.ContainerPort &&
				addr.ContainerPort <= MaxLocalRendezvousPort {
				addrs = append(addrs, addr)
			}
		}

		if len(addrs) == 1 {
			raddrs = append(raddrs, formatAddress(addrs[0]))
		} else {
			err = multierror.Append(err, fmt.Errorf(
				"found %d rendezvous addresses instead of 1 for container %s; dropping rendezvous addresses %v",
				len(addrs), caddr.id, addrs))
		}
	}
	return caddrs, raddrs, err.ErrorOrNil()
}

func formatAddress(p cproto.Address) string {
	return fmt.Sprintf("%s:%d", p.HostIP, p.HostPort)
}
