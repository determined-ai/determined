package api

import (
	"testing"
)

func TestPaginate(t *testing.T) {
	tables := []struct {
		Total       int
		Offset      int
		Limit       int
		ReturnCount int
		Err         bool
	}{
		{10, 0, 0, 10, false},
		{10, -3, 0, 3, false},
		{10, 3, 0, 7, false},
		{10, 13, 0, 7, true},
	}

	for _, tb := range tables {
		errReporter := func(msg string) {
			t.Errorf("Failed case: %v - %s", tb, msg)

		}
		p, err := Paginate(tb.Total, tb.Offset, tb.Limit)
		if tb.Err && err == nil || !tb.Err && err != nil {
			t.Errorf("failed test %v", tb) // TODO
			return
		}
		if err != nil {
			continue
		}

		retCount := p.EndIndex - p.StartIndex
		if p.StartIndex > p.EndIndex {
			errReporter("StartIndex after EndIndex")
		}
		if p.EndIndex < 0 || p.EndIndex > p.Total {
			errReporter("Endindex out of range")
		}
		if p.StartIndex < 0 || p.StartIndex > p.Total-1 {
			errReporter("StartIndex out of range")
		}
		if retCount != tb.ReturnCount {
			errReporter("Unexpected return count")
		}
	}
}
