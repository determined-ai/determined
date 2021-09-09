package task

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	cproto "github.com/determined-ai/determined/master/pkg/container"
)

type (
	// reservationWithState is an sproto.Reservation, along with its state
	// that is tracked by the allocation.
	reservationWithState struct {
		sproto.Reservation
		rank      int
		container *cproto.Container
		start     *sproto.TaskContainerStarted
		exit      *sproto.TaskContainerStopped
		daemon    bool
	}

	// reservations tracks reservations with their state.
	reservations map[cproto.ID]*reservationWithState
)

func newReservationState(r sproto.Reservation, rank int) reservationWithState {
	return reservationWithState{Reservation: r, rank: rank}
}

func (rs reservations) append(ars []sproto.Reservation) {
	start := len(rs)
	for rank, r := range ars {
		summary := r.Summary()
		state := newReservationState(r, start+rank)
		rs[summary.ID] = &state
	}
}

func (rs reservations) daemons() reservations {
	nrs := reservations{}
	for id, r := range rs {
		if r.daemon {
			nrs[id] = r
		}
	}
	return nrs
}

func (rs reservations) started() reservations {
	nrs := reservations{}
	for id, r := range rs {
		if r.start != nil {
			nrs[id] = r
		}
	}
	return nrs
}

func (rs reservations) exited() reservations {
	nrs := reservations{}
	for id, r := range rs {
		if r.exit != nil {
			nrs[id] = r
		}
	}
	return nrs
}
