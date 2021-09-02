package task

import (
	"fmt"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"gotest.tools/assert"

	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestRendezvous(t *testing.T) {
	const operations = 4
	type testCase struct {
		name  string
		order []int
	}

	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			// "task" with ranks is started.
			t1 := model.NewAllocationID(uuid.New().String())
			c1, c2 := cproto.NewID(), cproto.NewID()
			ranks := map[cproto.ID]int{c1: 0, c2: 1}
			r := NewRendezvous(t1, ranks)

			assert.Equal(t, r.rank(c1), 0)
			assert.Equal(t, r.rank(c2), 1)

			var ws []RendezvousWatcher
			watch := func(cID cproto.ID) func() {
				return func() {
					w, err := r.watch(t1, cID)
					assert.NilError(t, err, cID)
					ws = append(ws, w)
				}
			}

			startContainer := func(cID cproto.ID) func() {
				return func() {
					r.containerStarted(cID, addressesFromContainerID(cID))
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

			r.unwatch(c1)
			r.unwatch(c2)
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
	t1 := model.NewAllocationID(uuid.New().String())
	c1 := cproto.NewID()
	r := NewRendezvous(t1, map[cproto.ID]int{
		c1: 0,
	})

	_, err := r.watch(t1, cproto.NewID())
	assert.ErrorContains(t, err, "stale container")

	_, err = r.watch(t1, c1)
	assert.NilError(t, err)

	_, err = r.watch(t1, c1)
	assert.ErrorContains(t, err, "rendezvous request from already connected container")
}

func TestTerminationInRendezvous(t *testing.T) {
	t1 := model.NewAllocationID(uuid.New().String())
	c1, c2 := cproto.NewID(), cproto.NewID()
	ranks := map[cproto.ID]int{c1: 0, c2: 1}
	r := NewRendezvous(t1, ranks)

	r.containerStarted(c1, addressesFromContainerID(c1))
	_, err := r.watch(t1, c1)
	assert.NilError(t, err)
	r.containerTerminated(c1)

	r.containerStarted(c2, addressesFromContainerID(c2))
	_, err = r.watch(t1, c2)
	assert.NilError(t, err)

	assert.Check(t, !r.ready())
}

func TestUnwatchInRendezvous(t *testing.T) {
	t1 := model.NewAllocationID(uuid.New().String())
	c1, c2 := cproto.NewID(), cproto.NewID()
	ranks := map[cproto.ID]int{c1: 0, c2: 1}
	r := NewRendezvous(t1, ranks)

	r.containerStarted(c1, addressesFromContainerID(c1))
	_, err := r.watch(t1, c1)
	assert.NilError(t, err)
	r.unwatch(c1)

	r.containerStarted(c2, addressesFromContainerID(c2))
	_, err = r.watch(t1, c2)
	assert.NilError(t, err)

	assert.Check(t, !r.ready())
}

func TestRendezvousTimeout(t *testing.T) {
	RendezvousTimeoutDuration = 0

	t1 := model.NewAllocationID(uuid.New().String())
	c1, c2 := cproto.NewID(), cproto.NewID()
	ranks := map[cproto.ID]int{c1: 0, c2: 1}
	r := NewRendezvous(t1, ranks)

	_, err := r.watch(t1, c1)
	assert.NilError(t, err)
	r.containerStarted(c1, addressesFromContainerID(c1))

	time.Sleep(-1)
	assert.ErrorContains(t, r.checkTimeout(t1), "some containers are taking a long time")
}

func addressesFromContainerID(cID cproto.ID) []cproto.Address {
	return []cproto.Address{
		{
			ContainerIP:   "172.0.1.2",
			ContainerPort: 1734,
			HostIP:        fmt.Sprintf("%s.somehost.io", cID),
			HostPort:      1734,
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
					tmp := arr[i]
					arr[i] = arr[n-1]
					arr[n-1] = tmp
				} else {
					tmp := arr[0]
					arr[0] = arr[n-1]
					arr[n-1] = tmp
				}
			}
		}
	}

	helper(arr, len(arr))
	return res
}
