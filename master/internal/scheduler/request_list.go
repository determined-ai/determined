package scheduler

import (
	"strings"

	"github.com/determined-ai/determined/master/pkg/actor"

	"github.com/emirpasic/gods/sets/treeset"
)

// assignRequestList maintains assign requests in time order.
type assignRequestList struct {
	reqByTime    *treeset.Set
	reqByHandler map[*actor.Ref]*AssignRequest
	reqByID      map[RequestID]*AssignRequest
	assignments  map[*actor.Ref]*ResourceAssigned
}

func newAssignRequestList() *assignRequestList {
	return &assignRequestList{
		reqByTime:    treeset.NewWith(assignRequestComparator),
		reqByHandler: make(map[*actor.Ref]*AssignRequest),
		reqByID:      make(map[RequestID]*AssignRequest),
		assignments:  make(map[*actor.Ref]*ResourceAssigned),
	}
}

func (l *assignRequestList) iterator() *assignRequestIterator {
	return &assignRequestIterator{it: l.reqByTime.Iterator()}
}

func (l *assignRequestList) len() int {
	return len(l.reqByHandler)
}

func (l *assignRequestList) Get(handler *actor.Ref) (*AssignRequest, bool) {
	req, ok := l.reqByHandler[handler]
	return req, ok
}

func (l *assignRequestList) GetByID(id RequestID) (*AssignRequest, bool) {
	req, ok := l.reqByID[id]
	return req, ok
}

func (l *assignRequestList) Add(req *AssignRequest) bool {
	if _, ok := l.Get(req.Handler); ok {
		return false
	}

	l.reqByTime.Add(req)
	l.reqByHandler[req.Handler] = req
	l.reqByID[req.ID] = req
	return true
}

func (l *assignRequestList) Remove(handler *actor.Ref) *AssignRequest {
	req, ok := l.Get(handler)
	if !ok {
		return nil
	}

	l.reqByTime.Remove(req)
	delete(l.reqByHandler, handler)
	delete(l.reqByID, req.ID)
	delete(l.assignments, handler)
	return req
}

func (l *assignRequestList) GetAssignments(handler *actor.Ref) *ResourceAssigned {
	return l.assignments[handler]
}

func (l *assignRequestList) SetAssignments(handler *actor.Ref, assigned *ResourceAssigned) {
	l.assignments[handler] = assigned
}

func (l *assignRequestList) ClearAssignments(handler *actor.Ref) {
	delete(l.assignments, handler)
}

type assignRequestIterator struct{ it treeset.Iterator }

func (i *assignRequestIterator) next() bool            { return i.it.Next() }
func (i *assignRequestIterator) value() *AssignRequest { return i.it.Value().(*AssignRequest) }

func assignRequestComparator(a interface{}, b interface{}) int {
	t1, t2 := a.(*AssignRequest), b.(*AssignRequest)
	if t1.Handler.RegisteredTime().Equal(t2.Handler.RegisteredTime()) {
		return strings.Compare(string(t1.ID), string(t2.ID))
	}
	if t1.Handler.RegisteredTime().Before(t2.Handler.RegisteredTime()) {
		return -1
	}
	return 1
}
