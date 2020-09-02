package db

import (
	"fmt"
	"sync"

	"github.com/determined-ai/determined/master/pkg/etc"
)

type staticQueryMap struct {
	queries map[string]string
	sync.Mutex
}

func (q *staticQueryMap) getOrLoad(queryName string) string {
	q.Lock()
	defer q.Unlock()
	query, ok := q.queries[queryName]
	if !ok {
		query = string(etc.MustStaticFile(fmt.Sprintf("%s.sql", queryName)))
		q.queries[queryName] = query
	}
	return query
}
