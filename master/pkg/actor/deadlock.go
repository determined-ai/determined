package actor

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/topo"
)

// DeadlockDetectorEnabled configures actor systems to detect ask deadlocks
// at runtime.
var DeadlockDetectorEnabled = false

// detectDeadlock checks the current state of asks in the system to determine
// if a deadlock has occurred. As response.get() is called after an ask,
// an edge is added to the graph of asks (system.asks). Then the graph
// is checked for cycles.
//
// A directed cycle of asks necessarily indicates a deadlock; each node in the
// cycle is asking another node that will never finish its own ask.
//
// If a deadlock is detected, the cycle is printed. If this is not enough information,
// the gonum also provides libraries to dump the current asks in DOT format, to
// view with graphviz.
func detectDeadlock(system *System, sender, receiver Address) func() {
	g := system.asks
	g.Lock()
	defer g.Unlock()
	if sender == receiver {
		// If this is a self ask, it will deadlock that actor so we print a
		// warning, but we also just exit because gonum's graph will panic.
		logSelfAskWarning(receiver)
		return func() {}
	}
	g.SetEdge(g.NewEdge(sender, receiver))
	if cycles := topo.DirectedCyclesIn(g); len(cycles) > 0 {
		logDeadlockWarnings(cycles)
	}
	return func() {
		g.Lock()
		defer g.Unlock()
		g.RemoveEdge(sender.ID(), receiver.ID())
	}
}

func closeDeadlockDetectorResources(r *Ref) {
	r.system.asks.RemoveNode(r.Address().ID())
}

func logSelfAskWarning(self Address) {
	logrus.Warnf("self ask detected %s", self)
}

func logDeadlockWarnings(cycles [][]graph.Node) {
	warning := "potential actor deadlocks (ask cycle) detected"
	for i, cycle := range cycles {
		warning += fmt.Sprintf("\n  %dth deadlock", i)
		for _, node := range cycle {
			warning += fmt.Sprintf("\n    %s awaiting", node.(Address))
		}
	}
	logrus.Warn(warning)
}
