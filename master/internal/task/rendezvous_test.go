package task

import (
	"fmt"
	"testing"

	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/taskmodel"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestRendezvous(t *testing.T) {
	const operations = 4
	type testCase struct {
		name  string
		order []int
	}

	res := mocks.NewResources(t)
	res.On("Summary").Return(sproto.ResourcesSummary{
		AgentDevices: map[aproto.ID][]device.Device{},
	})

	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			// "task" with ranks is started.
			t1 := model.AllocationID(uuid.New().String())
			c1, c2 := sproto.ResourcesID(cproto.NewID()), sproto.ResourcesID(cproto.NewID())
			r := newRendezvous(nil, t1, resourcesList{
				c1: &taskmodel.ResourcesWithState{
					Resources: res,
					Rank:      0,
				},
				c2: &taskmodel.ResourcesWithState{
					Resources: res,
					Rank:      1,
				},
			})

			var ws []RendezvousWatcher
			watch := func(rID sproto.ResourcesID) func() {
				return func() {
					w, err := r.watch(WatchRendezvousInfo{ResourcesID: rID})
					assert.NilError(t, err, rID)
					ws = append(ws, w)
				}
			}

			startContainer := func(rID sproto.ResourcesID) func() {
				return func() {
					r.resources[rID].Started = &sproto.ResourcesStarted{
						Addresses: addressesFromContainerID(rID),
					}
					r.try()
				}
			}

			ops := []func(){
				watch(c1),
				watch(c2),
				startContainer(c1),
				startContainer(c2),
			}
			for _, i := range tc.order {
				assert.Check(t, !r.ready())
				ops[i]()
			}
			assert.Check(t, r.ready())

			rendezvousArrived := func(w RendezvousWatcher) {
				select {
				case resp := <-w.C:
					assert.NilError(t, resp.Err)
					assert.Equal(t, len(resp.Info.Addresses), 2)
				default:
					t.Fatal("expected rendezvous on first watcher but found none")
				}
			}
			for _, w := range ws {
				rendezvousArrived(w)
			}

			r.unwatch(UnwatchRendezvousInfo{ResourcesID: c1})
			r.unwatch(UnwatchRendezvousInfo{ResourcesID: c2})
		})
	}

	for _, ordering := range orderings(operations) {
		runTestCase(t, testCase{
			name:  spew.Sdump(ordering),
			order: ordering,
		})
	}
}

func TestRendezvousValidation(t *testing.T) {
	t1 := model.AllocationID(uuid.New().String())
	c1 := sproto.ResourcesID(cproto.NewID())
	r := newRendezvous(nil, t1, resourcesList{
		c1: &taskmodel.ResourcesWithState{Rank: 0},
	})

	_, err := r.watch(WatchRendezvousInfo{ResourcesID: sproto.ResourcesID(cproto.NewID())})
	assert.ErrorContains(t, err, "stale resources")

	_, err = r.watch(WatchRendezvousInfo{ResourcesID: c1})
	assert.NilError(t, err)

	_, err = r.watch(WatchRendezvousInfo{ResourcesID: c1})
	assert.ErrorContains(t, err, "resources already rendezvoused")
}

func TestTerminationInRendezvous(t *testing.T) {
	t1 := model.AllocationID(uuid.New().String())
	c1, c2 := sproto.ResourcesID(cproto.NewID()), sproto.ResourcesID(cproto.NewID())
	r := newRendezvous(nil, t1, resourcesList{
		c1: &taskmodel.ResourcesWithState{Rank: 0},
		c2: &taskmodel.ResourcesWithState{Rank: 1},
	})

	r.resources[c1].Started = &sproto.ResourcesStarted{
		Addresses: addressesFromContainerID(c1),
	}
	r.try()
	_, err := r.watch(WatchRendezvousInfo{ResourcesID: c1})
	assert.NilError(t, err)
	r.resources[c1].Exited = &sproto.ResourcesStopped{}

	r.resources[c2].Started = &sproto.ResourcesStarted{
		Addresses: addressesFromContainerID(c2),
	}
	r.try()
	_, err = r.watch(WatchRendezvousInfo{ResourcesID: c2})
	assert.NilError(t, err)

	assert.Check(t, !r.ready())
}

func TestUnwatchInRendezvous(t *testing.T) {
	t1 := model.AllocationID(uuid.New().String())
	c1, c2 := sproto.ResourcesID(cproto.NewID()), sproto.ResourcesID(cproto.NewID())
	r := newRendezvous(nil, t1, resourcesList{
		c1: &taskmodel.ResourcesWithState{Rank: 0},
		c2: &taskmodel.ResourcesWithState{Rank: 1},
	})

	r.resources[c1].Started = &sproto.ResourcesStarted{Addresses: addressesFromContainerID(c1)}
	r.try()
	_, err := r.watch(WatchRendezvousInfo{ResourcesID: c1})
	assert.NilError(t, err)
	r.unwatch(UnwatchRendezvousInfo{ResourcesID: c1})

	r.resources[c2].Started = &sproto.ResourcesStarted{Addresses: addressesFromContainerID(c2)}
	r.try()
	_, err = r.watch(WatchRendezvousInfo{ResourcesID: c2})
	assert.NilError(t, err)

	assert.Check(t, !r.ready())
}

func TestRendezvousTimeout(t *testing.T) {
	rendezvousTimeoutDuration = 0

	t1 := model.AllocationID(uuid.New().String())
	c1, c2 := sproto.ResourcesID(cproto.NewID()), sproto.ResourcesID(cproto.NewID())
	r := newRendezvous(nil, t1, resourcesList{
		c1: &taskmodel.ResourcesWithState{Rank: 0},
		c2: &taskmodel.ResourcesWithState{Rank: 1},
	})

	_, err := r.watch(WatchRendezvousInfo{ResourcesID: c1})
	assert.NilError(t, err)
	r.resources[c1].Started = &sproto.ResourcesStarted{Addresses: addressesFromContainerID(c1)}
	r.try()

	assert.ErrorContains(t, r.checkTimeout(rendezvousTimeout{AllocationID: t1}),
		"some containers are taking a long time")
}

func addressesFromContainerID(rID sproto.ResourcesID) []cproto.Address {
	hostIP := fmt.Sprintf("%s.example.com", rID)
	hostPort := 1734

	return []cproto.Address{
		{
			ContainerIP:   "172.0.1.2",
			ContainerPort: 1734,
			HostIP:        &hostIP,
			HostPort:      &hostPort,
		},
	}
}

// orderings returns all orders for n operations.
func orderings(n int) [][]int {
	var xs []int
	for i := 0; i < n; i++ {
		xs = append(xs, i)
	}
	return permutations(xs)
}

// https://stackoverflow.com/questions/30226438/generate-all-permutations-in-go
func permutations(arr []int) [][]int {
	var helper func([]int, int)
	res := [][]int{}

	helper = func(arr []int, n int) {
		if n == 1 {
			tmp := make([]int, len(arr))
			copy(tmp, arr)
			res = append(res, tmp)
		} else {
			for i := 0; i < n; i++ {
				helper(arr, n-1)
				if n%2 == 1 {
					arr[i], arr[n-1] = arr[n-1], arr[i]
				} else {
					arr[0], arr[n-1] = arr[n-1], arr[0]
				}
			}
		}
	}

	helper(arr, len(arr))
	return res
}
