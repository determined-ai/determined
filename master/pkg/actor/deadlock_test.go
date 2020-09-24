package actor

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/logger"
)

func TestDeadlockDetector(t *testing.T) {
	DeadlockDetectorEnabled = true

	type testCase struct {
		name         string
		setup        func(s *System)
		shouldDetect bool
	}

	tests := []testCase{
		{
			name: "simple deadlock single chain",
			setup: func(s *System) {
				ref1, _ := mockForwarder(s, Addr("a1"))
				ref2, _ := mockForwarder(s, Addr("a2"))
				s.Tell(ref1, []instruction{
					{ref2, 0},
					{ref1, 0},
				})
			},
			shouldDetect: true,
		},
		{
			name: "simple deadlock multiple chain",
			setup: func(s *System) {
				ref1, _ := mockForwarder(s, Addr("a1"))
				ref2, _ := mockForwarder(s, Addr("a2"))
				// Pause each to make sure they both get into their receive
				s.Tell(ref1, []instruction{
					{ref2, time.Millisecond * 5},
				})
				s.Tell(ref2, []instruction{
					{ref1, time.Millisecond * 5},
				})
			},
			shouldDetect: true,
		},
		{
			name: "complex deadlock single chains",
			setup: func(s *System) {
				ref1, _ := mockForwarder(s, Addr("a1"))
				ref2, _ := mockForwarder(s, Addr("a2"))
				ref3, _ := mockForwarder(s, Addr("a3"))
				ref4, _ := mockForwarder(s, Addr("a4"))
				ref5, _ := mockForwarder(s, Addr("a5"))
				s.Tell(ref1, []instruction{
					{ref2, 0},
					{ref3, 0},
					{ref4, 0},
					{ref5, 0},
					{ref3, 0},
				})
			},
			shouldDetect: true,
		},
		{
			name: "complex deadlock multiple chains",
			setup: func(s *System) {
				ref1, _ := mockForwarder(s, Addr("a1"))
				ref2, _ := mockForwarder(s, Addr("a2"))
				ref3, _ := mockForwarder(s, Addr("a3"))
				ref4, _ := mockForwarder(s, Addr("a4"))
				ref5, _ := mockForwarder(s, Addr("a5"))
				// "Illustration" of what's happening (or supposed to)
				// Start a chain from a1, a2, a3, a4, a5 and a4, a3, a2:
				//  a1 ---> a2 ---> a3   (a2 makes it to a3 first, but a4 has already started its ask)
				//                  ^ <--- a4
				// They meet in the middle and get stuck.
				//  a1 ---> a2 ---> a3 ---> a4
				//                  ^-------v
				s.Tell(ref1, []instruction{
					{ref2, time.Millisecond * 5},
					{ref3, 0},
					{ref4, 0},
					{ref5, 0},
				})
				s.Tell(ref4, []instruction{
					{ref3, time.Millisecond * 10},
					{ref2, 0},
				})
			},
			shouldDetect: true,
		},
		{
			name: "simple no deadlock",
			setup: func(s *System) {
				ref1, _ := mockForwarder(s, Addr("a1"))
				ref2, _ := mockForwarder(s, Addr("a2"))
				s.Tell(ref2, []instruction{{ref1, 0}})
			},
			shouldDetect: false,
		},
		{
			name: "complex deadlock single chains",
			setup: func(s *System) {
				ref1, _ := mockForwarder(s, Addr("a1"))
				ref2, _ := mockForwarder(s, Addr("a2"))
				ref3, _ := mockForwarder(s, Addr("a3"))
				ref4, _ := mockForwarder(s, Addr("a4"))
				ref5, _ := mockForwarder(s, Addr("a5"))
				s.Tell(ref1, []instruction{
					{ref2, 0},
					{ref3, 0},
					{ref4, 0},
					{ref5, 0},
					{ref3, 0},
				})
			},
			shouldDetect: true,
		},
	}

	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			logStore := logger.NewLogBuffer(100)
			logrus.AddHook(logStore)
			s := NewSystem("")
			tc.setup(s)
			time.Sleep(time.Millisecond * 100)
			if tc.shouldDetect {
				assertLogsContain(t, logStore, "actor deadlock", "deadlock not detected")
			} else {
				assertLogsDoNotContain(t, logStore, "actor deadlock", "deadlock improperly detected")
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func assertLogsContain(t *testing.T, logStore *logger.LogBuffer, expected string, msg string) {
	for _, entry := range logStore.Entries(0, 100, 100) {
		fmt.Println(entry.Message)
		if strings.Contains(entry.Message, expected) {
			return
		}
	}
	t.Error(msg)
	t.Fail()
}

func assertLogsDoNotContain(t *testing.T, logStore *logger.LogBuffer, expected string, msg string) {
	for _, entry := range logStore.Entries(0, 100, 100) {
		fmt.Println(entry.Message)
		if strings.Contains(entry.Message, expected) {
			t.Error(msg)
			t.Fail()
		}
	}
}

type instruction struct {
	forwardTo *Ref
	pause     time.Duration
}

func mockForwarder(system *System, address Address) (*Ref, bool) {
	return system.ActorOf(address, ActorFunc(func(context *Context) error {
		if msg, ok := context.Message().([]instruction); ok && len(msg) > 0 {
			instr, rest := msg[0], msg[1:]
			time.Sleep(instr.pause)
			context.Ask(instr.forwardTo, rest).Get()
		}
		if context.ExpectingResponse() {
			context.Respond(context.Message())
		}
		if err, ok := context.Message().(error); ok {
			return err
		}
		return nil
	}))
}
