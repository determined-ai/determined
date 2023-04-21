package task

import (
	"fmt"
	"sort"
	"time"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	apiutils "github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

const (
	// minLocalRendezvousPort is the smallest port to use (from the container's point of view;
	// it will be mapped to some arbitrary port on the host) for communication across containers.
	minLocalRendezvousPort = 1734

	// maxLocalRendezvousPort is the largest port to use for communication across containers.
	// Each distributed trial can take up to 2 host based ports and we assume a maximum.
	// of 16 slot per agent. maxLocalRendezvousPort = MinLocalRendezvousPort + 2*16 - 1.
	maxLocalRendezvousPort = minLocalRendezvousPort + 2*16 - 1
)

// rendezvousTimeoutDuration is the default timeout for rendezvous.
var rendezvousTimeoutDuration = 10 * time.Minute

type (
	// WatchRendezvousInfo begins watching for rendezvous info.
	// When all the containers are ready, the trial will send all the
	// peer addresses on the channel in the response.
	WatchRendezvousInfo struct {
		ResourcesID sproto.ResourcesID
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
	UnwatchRendezvousInfo struct {
		ResourcesID sproto.ResourcesID
	}

	// rendezvousTimeout tracks the timeout of the allocation resources rendezvousing.
	// It is possible that it takes very long for all containers to be connected after the first
	// container is connected. This might happen when the k8s cluster waits for new instances
	// to spin up, which might not happen at all. At the same time, taking up part of all
	// the resources and waiting is wasteful. So we need to detect this situation.
	rendezvousTimeout struct{ AllocationID model.AllocationID }

	// rendezvous encapsulates the rendezvous state of a trial.
	rendezvous struct {
		allocationID      model.AllocationID
		watchers          map[sproto.ResourcesID]chan<- RendezvousInfoOrError
		resources         resourcesList
		lastWatchTime     time.Time
		allReadySucceeded bool
	}
)

// newRendezvous returns a new rendezvous component.
func newRendezvous(
	ctx *actor.Context,
	allocationID model.AllocationID,
	rs resourcesList,
) *rendezvous {
	if ctx != nil {
		actors.NotifyAfter(ctx, rendezvousTimeoutDuration, rendezvousTimeout{
			AllocationID: allocationID,
		})
	}

	return &rendezvous{
		allocationID: allocationID,
		resources:    rs,
		watchers:     map[sproto.ResourcesID]chan<- RendezvousInfoOrError{},
	}
}

func (r *rendezvous) watch(msg WatchRendezvousInfo) (RendezvousWatcher, error) {
	if _, ok := r.resources[msg.ResourcesID]; !ok {
		err := ErrStaleResources{ID: msg.ResourcesID}
		return RendezvousWatcher{}, apiutils.AsValidationError(err.Error())
	} else if _, ok := r.watchers[msg.ResourcesID]; ok {
		return RendezvousWatcher{}, apiutils.AsValidationError(
			"resources already rendezvoused: %s", msg.ResourcesID,
		)
	}

	// Channel is size 1 since rendezvous info will only ever be sent once.
	w := make(chan RendezvousInfoOrError, 1)
	r.watchers[msg.ResourcesID] = w
	r.lastWatchTime = time.Now()
	if r.ready() {
		r.push()
	}
	return RendezvousWatcher{C: w}, nil
}

func (r *rendezvous) unwatch(msg UnwatchRendezvousInfo) {
	if r == nil {
		return
	}
	delete(r.watchers, msg.ResourcesID)
}

func (r *rendezvous) try() bool {
	if r.ready() {
		r.push()
	}
	return r.ready()
}

// ready returns true if and only if all the containers are reported to be started with the
// ContainerStarted message and their sockets to be connected with the containerConnected
// message. The two messages are not guaranteed to come in-order. During each run of the
// trial, once all the containers are ready this function will return true afterward because this
// function is used in deciding if the trial should be forcibly killed when terminating.
func (r *rendezvous) ready() bool {
	if r == nil {
		return false
	}

	// If a trial has passed allReady it can never return to a state of not ready until the
	// current containers are all taskTerminated.
	if r.allReadySucceeded {
		return true
	}

	anyExited := len(r.resources.exited()) > 0
	allAddressesArrived := len(r.resources.started()) == len(r.resources)
	allWaiting := len(r.watchers) == len(r.resources)

	r.allReadySucceeded = !anyExited && allAddressesArrived && allWaiting
	return r.allReadySucceeded
}

// push gathers up the external addresses for the exposed ports and sends them to all the
// containers in the trial.
func (r rendezvous) push() bool {
	if !r.ready() {
		return false
	}
	caddrs, raddrs, slotCounts, err := r.info()
	for _, caddr := range caddrs {
		w := r.watchers[caddr.id]
		w <- RendezvousInfoOrError{
			Info: &trialv1.RendezvousInfo{
				Addresses: raddrs,
				Slots:     slotCounts,
				Rank:      int32(r.resources[caddr.id].Rank),
			},
			Err: err,
		}
		close(w)
		delete(r.watchers, caddr.id)
	}
	return true
}

// checkTimeout checks if the task should timeout waiting for rendezvous.
func (r *rendezvous) checkTimeout(msg rendezvousTimeout) error {
	if r == nil || r.allReadySucceeded {
		return nil
	}

	exceededTimeout := time.Now().After(r.lastWatchTime.Add(rendezvousTimeoutDuration))
	if r.allocationID == msg.AllocationID && exceededTimeout {
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

// close closes rendezvous by letting still active watchers know they were terminated.
func (r *rendezvous) close() {
	if r == nil {
		return
	}

	for cID, w := range r.watchers {
		w <- RendezvousInfoOrError{Err: errors.New("task terminated")}
		close(w)
		delete(r.watchers, cID)
	}
}

type cAddress struct {
	id        sproto.ResourcesID
	addresses []cproto.Address
	ordinal   int
	slots     int
}

func (r *rendezvous) info() ([]cAddress, []string, []int32, error) {
	var caddrs []cAddress
	for id, r := range r.resources {
		caddr := cAddress{
			id:        id,
			addresses: r.Started.Addresses,
			ordinal:   r.Rank,
			slots:     r.Summary().Slots(),
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
	var slots []int32
	var err *multierror.Error
	for _, caddr := range caddrs {
		var addrs []cproto.Address
		for _, addr := range caddr.addresses {
			if minLocalRendezvousPort <= addr.ContainerPort &&
				addr.ContainerPort <= maxLocalRendezvousPort {
				addrs = append(addrs, addr)
			}
		}

		if len(addrs) == 1 {
			raddrs = append(raddrs, addrs[0].TargetIP())
			slots = append(slots, int32(caddr.slots))
		} else {
			err = multierror.Append(err, fmt.Errorf(
				"found %d rendezvous addresses instead of 1 for container %s; dropping rendezvous addresses %v",
				len(addrs), caddr.id, addrs))
		}
	}
	return caddrs, raddrs, slots, err.ErrorOrNil()
}
