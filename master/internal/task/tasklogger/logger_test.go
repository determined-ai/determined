package tasklogger_test

import (
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/task/tasklogger"
	"github.com/determined-ai/determined/master/pkg/model"
)

type arrayWriter struct {
	t            *testing.T
	mu           sync.Mutex
	logs         []*model.TaskLog
	nextFlushErr error
}

// AddTaskLogs implements tasklogger.Writer.
func (aw *arrayWriter) AddTaskLogs(logs []*model.TaskLog) error {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	if aw.nextFlushErr != nil {
		tmp := aw.nextFlushErr
		aw.nextFlushErr = nil
		return tmp
	}

	aw.t.Logf("writing %d logs", len(logs))
	aw.logs = append(aw.logs, logs...)
	return nil
}

func (aw *arrayWriter) readLogs() []*model.TaskLog {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	aw.t.Logf("reading %d logs", len(aw.logs))
	logs := aw.logs
	aw.logs = nil
	return logs
}

func (aw *arrayWriter) getNextFlushErr() error {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	return aw.nextFlushErr
}

func (aw *arrayWriter) setNextFlushErr(err error) {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	aw.nextFlushErr = err
}

func TestTaskLoggerFailures(t *testing.T) {
	w := &arrayWriter{t: t}
	tasklogger.SetDefaultLogger(tasklogger.New(w))

	// With a successful backend, logs should be flushed.
	fakeLog := &model.TaskLog{Log: "test"}
	tasklogger.Insert(fakeLog)
	waitForCondition(t, time.Second, func() bool {
		return slices.Contains(w.readLogs(), fakeLog)
	})

	// On error, logs should be dropped.
	w.setNextFlushErr(fmt.Errorf("test"))
	tasklogger.Insert(fakeLog)
	waitForCondition(t, time.Second, func() bool {
		return w.getNextFlushErr() == nil
	})

	// Later logs should still be flushed (by omission, checks fakeLog1 isn't flushed).
	fakeLog2 := &model.TaskLog{Log: "test2"}
	tasklogger.Insert(fakeLog2)
	waitForCondition(t, time.Second, func() bool {
		return slices.Contains(w.readLogs(), fakeLog2)
	})
}

func TestTaskLoggerNil(t *testing.T) {
	// Just test this doesn't panic, since the actor system also used to not panic.
	tasklogger.SetDefaultLogger(nil)
	tasklogger.Insert(&model.TaskLog{})
}

func TestTaskLoggerConcurrent(t *testing.T) {
	w := &arrayWriter{t: t}
	tasklogger.SetDefaultLogger(tasklogger.New(w))

	var in []*model.TaskLog
	for i := 0; i < 10000; i++ {
		i := i
		in = append(in, &model.TaskLog{ID: &i})
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	var idx int
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				mu.Lock()
				if idx >= len(in) {
					mu.Unlock()
					return
				}
				log := in[idx]
				idx++
				mu.Unlock()

				tasklogger.Insert(log)
			}
		}()
	}
	wg.Wait()

	var written []*model.TaskLog
	waitForCondition(t, time.Minute,
		func() bool {
			written = append(written, w.readLogs()...)
			if len(written) >= len(in) {
				t.Log("all logs written")
				return true
			}
			return false
		},
	)
	require.Equal(t, len(written), len(in))
	require.ElementsMatch(t, written, in)
}

func waitForCondition(
	t *testing.T,
	timeout time.Duration,
	condition func() bool,
) {
	for i := 0; i < int(timeout/tasklogger.FlushInterval); i++ {
		if condition() {
			return
		}
		time.Sleep(tasklogger.FlushInterval)
	}
}
