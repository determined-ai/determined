package db

import (
	"fmt"
	"sync"

	"github.com/determined-ai/determined/master/pkg/etc"
)

type staticQueryMap struct {
	queries map[string]string
	sync.RWMutex
}

func (q *staticQueryMap) getOrLoad(queryName string) string {
	q.RLock()
	query, ok := q.queries[queryName]
	q.RUnlock()
	if !ok {
		query = string(etc.MustStaticFile(fmt.Sprintf("%s.sql", queryName)))
		q.Lock()
		q.queries[queryName] = query
		q.Unlock()
	}
	return query
}
