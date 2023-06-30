package allgather_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/task/allgather"
)

func TestAllGather(t *testing.T) {
	groupID := uuid.NewString()
	numPeers := 4

	var ready atomic.Bool
	readyFunc := func() {
		ready.Store(true)
	}

	var timedOut atomic.Bool
	timeoutFunc := func(error) {
		timedOut.Store(true)
	}

	var watchers []allgather.Watcher
	var expectedResult []any
	for i := 0; i < numPeers; i++ {
		procID := uuid.New()
		expectedResult = append(expectedResult, procID.String())

		w := allgather.Join(
			groupID,
			procID,
			numPeers,
			procID.String(),
			readyFunc,
			timeoutFunc,
		)
		defer allgather.Leave(groupID, procID)
		watchers = append(watchers, w)
	}

	for _, w := range watchers {
		res := <-w.C
		require.NoError(t, res.Err)
		require.ElementsMatch(t, expectedResult, res.Data)
	}
	require.True(t, ready.Load())
	require.False(t, timedOut.Load())
}

func TestAllGatherRejoin(t *testing.T) {
	groupID := uuid.NewString()
	numPeers := 4

	var ready atomic.Bool
	readyFunc := func() {
		ready.Store(true)
	}

	var timedOut atomic.Bool
	timeoutFunc := func(error) {
		timedOut.Store(true)
	}

	// All but two of the watchers connect immediately.
	var watchers []allgather.Watcher
	var expectedResult []string
	for i := 0; i < numPeers-2; i++ {
		procID := uuid.New()
		expectedResult = append(expectedResult, procID.String())

		w := allgather.Join(
			groupID,
			procID,
			numPeers,
			procID.String(),
			readyFunc,
			timeoutFunc,
		)
		defer allgather.Leave(groupID, procID)
		watchers = append(watchers, w)
	}

	// One joins once and leave.
	procID := uuid.New()
	expectedResult = append(expectedResult, procID.String())
	_ = allgather.Join(
		groupID,
		procID,
		numPeers,
		procID.String(),
		readyFunc,
		timeoutFunc,
	)
	allgather.Leave(groupID, procID)

	// Then joins for real, but we shouldn't be ready.
	rejoiner := allgather.Join(
		groupID,
		procID,
		numPeers,
		procID.String(),
		readyFunc,
		timeoutFunc,
	)
	defer allgather.Leave(groupID, procID)
	require.False(t, ready.Load())
	require.False(t, timedOut.Load())

	// By some error it also joins again, but we still shouldn't be ready.
	w := allgather.Join(
		groupID,
		procID,
		numPeers,
		procID.String(),
		readyFunc,
		timeoutFunc,
	)
	defer allgather.Leave(groupID, procID)
	watchers = append(watchers, w)
	require.False(t, ready.Load())
	require.False(t, timedOut.Load())

	// Final watcher joins.
	procID = uuid.New()
	expectedResult = append(expectedResult, procID.String())
	w = allgather.Join(
		groupID,
		procID,
		numPeers,
		procID.String(),
		readyFunc,
		timeoutFunc,
	)
	defer allgather.Leave(groupID, procID)
	watchers = append(watchers, w)

	for _, w := range watchers {
		res := <-w.C
		require.NoError(t, res.Err)
		require.ElementsMatch(t, expectedResult, res.Data)
	}
	res := <-rejoiner.C
	require.ErrorIs(t, res.Err, allgather.ErrReconnected)

	require.True(t, ready.Load())
	require.False(t, timedOut.Load())
}

func TestAllGatherTimeout(t *testing.T) {
	defaultTimeout := allgather.DefaultTimeout
	defer func() { allgather.DefaultTimeout = defaultTimeout }()
	allgather.DefaultTimeout = time.Second

	groupID := uuid.NewString()
	numPeers := 4

	var ready atomic.Bool
	readyFunc := func() {
		ready.Store(true)
	}

	var timedOut atomic.Bool
	timeoutFunc := func(error) {
		timedOut.Store(true)
	}

	var watchers []allgather.Watcher
	for i := 0; i < numPeers/2; i++ {
		procID := uuid.New()
		w := allgather.Join(
			groupID,
			procID,
			numPeers,
			procID.String(),
			readyFunc,
			timeoutFunc,
		)
		defer allgather.Leave(groupID, procID)
		watchers = append(watchers, w)
	}

	require.True(t, waitForCondition(5*time.Second, timedOut.Load))
	require.False(t, ready.Load())
	for _, w := range watchers {
		select {
		case <-w.C:
			require.Fail(t, "timed-out, incomplete watcher should have no data")
		default:
		}
	}
}

var tickInterval = 100 * time.Millisecond

func waitForCondition(timeout time.Duration, condition func() bool) bool {
	for i := 0; i < int(timeout/tickInterval); i++ {
		if condition() {
			return true
		}
		time.Sleep(tickInterval)
	}
	return false
}
