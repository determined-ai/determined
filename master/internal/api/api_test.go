package api

import (
	"testing"
)

func TestPaginate(t *testing.T) {
	tables := []struct {
		Total       int
		Offset      int
		Limit       int
		StartIndex  int
		ReturnCount int
		Err         bool
	}{
		{10, 0, 0, 0, 10, false},
		{10, 1, 0, 1, 9, false},
		{10, 1, 1, 1, 1, false},
		{10, -1, 0, 9, 1, false},
		{10, -2, 1, 8, 1, false},
		{10, 13, 0, 0, 7, true},
	}

	for _, tb := range tables {
		tb := tb
		errReporter := func(msg string) {
			t.Errorf("Failed case: %v - %s", tb, msg)
		}
		p, err := Paginate(tb.Total, tb.Offset, tb.Limit)
		if tb.Err && err == nil || !tb.Err && err != nil {
			t.Errorf("failed test %v", tb)
			return
		}
		if err != nil {
			continue
		}

		if p.StartIndex > p.EndIndex {
			errReporter("StartIndex after EndIndex")
		}
		if p.EndIndex < 0 || p.EndIndex > tb.Total {
			errReporter("Endindex out of range")
		}
		if p.StartIndex < 0 || p.StartIndex > tb.Total-1 {
			errReporter("StartIndex out of range")
		}
		if p.StartIndex != tb.StartIndex {
			errReporter("Unexpected start index")
		}
		retCount := p.EndIndex - p.StartIndex
		if retCount != tb.ReturnCount {
			errReporter("Unexpected return count")
		}
	}
}
