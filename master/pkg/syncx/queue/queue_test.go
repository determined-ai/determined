package queue_test

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/syncx/queue"
)

func TestQueue(t *testing.T) {
	q := queue.New[int]()
	require.Equal(t, 0, q.Len())

	q.Put(1)
	require.Equal(t, 1, q.Len())

	q.Put(2)
	require.Equal(t, 2, q.Len())

	require.Equal(t, 1, q.Get())
	require.Equal(t, 1, q.Len())

	require.Equal(t, 2, q.Get())
	require.Equal(t, 0, q.Len())

	_, ok := q.TryGet()
	require.False(t, ok)
	require.Equal(t, 0, q.Len())

	done := make(chan struct{})
	go func() {
		require.Equal(t, 3, q.Get())
		close(done)
	}()

	select {
	case <-time.NewTimer(100 * time.Millisecond).C:
	case <-done:
		require.FailNow(t, "get should have blocked")
	}

	q.Put(3)

	select {
	case <-time.NewTimer(100 * time.Millisecond).C:
		require.FailNow(t, "get should have unblocked")
	case <-done:
	}

	require.Equal(t, 0, q.Len())
}

func TestQueueMultipleBlockedReaders(t *testing.T) {
	t.Log("creating queue with max size 1")
	q := queue.New[int]()
	require.Equal(t, 0, q.Len())

	t.Log("launch goroutines to add 3 elements")
	var mu sync.Mutex
	in := []int{0, 1, 2, 3, 4, 5}
	var out []int
	var dones []chan struct{}
	for _, i := range in {
		done := make(chan struct{})
		dones = append(dones, done)
		go func(i int) {
			tmp := q.Get()

			mu.Lock()
			defer mu.Unlock()
			out = append(out, tmp)
			close(done)
		}(i)
	}

	time.Sleep(100 * time.Millisecond)

	t.Log("check they are all blocked")
	for _, done := range dones {
		select {
		case <-done:
			require.FailNow(t, "put should have blocked")
		default:
		}
	}

	t.Log("add 1 element")
	q.Put(in[0])
	time.Sleep(100 * time.Millisecond)

	t.Log("check that exactly 1 unblocks")
	var numDone int
	for _, done := range dones {
		select {
		case <-done:
			numDone++
		default:
		}
	}
	require.Equal(t, 1, numDone, "only one put should have unblocked")

	t.Log("add the rest")
	for _, i := range in[1:] {
		q.Put(i)
	}
	time.Sleep(100 * time.Millisecond)

	t.Log("check that all goroutines unblock")
	numDone = 0
	for _, done := range dones {
		select {
		case <-done:
			numDone++
		default:
		}
	}
	require.Equal(t, len(in), numDone, "all goroutines should have unblocked")

	require.ElementsMatch(t, in, out, "should have gotten all values")
	require.Equal(t, 0, q.Len())
}

func TestQueueConcurrent(t *testing.T) {
	q := queue.New[int]()
	require.Equal(t, 0, q.Len())

	var in []int
	for i := 0; i < 100; i++ {
		in = append(in, i)
	}
	rand.Shuffle(len(in), func(i, j int) { in[i], in[j] = in[j], in[i] })

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, i := range in {
			q.Put(i)
		}
	}()

	var out []int

	wg.Add(1)
	go func() {
		defer wg.Done()

		for range in {
			out = append(out, q.Get())
		}
	}()

	wg.Wait()
	require.Equal(t, in, out)
}
