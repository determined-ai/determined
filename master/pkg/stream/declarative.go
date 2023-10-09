package stream

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type keySetIter struct {
	vals  []string
	idx   int
	start int64
	end   int64
}

func newKeySetIter(keyset string) *keySetIter {
	vals := strings.Split(keyset, ",")
	// handle empty string case
	if len(vals) == 1 && len(vals[0]) == 0 {
		return &keySetIter{vals: nil}
	}
	return &keySetIter{vals: strings.Split(keyset, ",")}
}

// Next returns a tuple of (ok, value, err).
func (ksi *keySetIter) Next() (bool, int64, error) {
	// are we emitting from a range?
	if ksi.start < ksi.end {
		out := ksi.start
		ksi.start++
		return true, out, nil
	}
	// are we out of values?
	if ksi.idx >= len(ksi.vals) {
		return false, 0, nil
	}
	// parse the next value
	val := ksi.vals[ksi.idx]
	ksi.idx++
	subs := strings.Split(val, "-")
	if len(subs) == 1 {
		// it's a single value
		n, err := strconv.ParseInt(subs[0], 10, 64)
		if err != nil {
			return false, 0, errors.Wrapf(err, "invalid number (%v)", val)
		}
		return true, n, nil
	} else if len(subs) == 2 {
		// it's a range
		start, err := strconv.ParseInt(subs[0], 10, 64)
		if err != nil {
			return false, 0, errors.Wrapf(err, "invalid range (%v)", val)
		}
		end, err := strconv.ParseInt(subs[1], 10, 64)
		if err != nil {
			return false, 0, errors.Wrapf(err, "invalid range (%v)", val)
		}
		if start >= end {
			return false, 0, errors.Errorf("invalid range (%v)", val)
		}
		ksi.start = start + 1 // plus one because we emit start right now
		ksi.end = end + 1     // plus one to save an exclusive endpoint
		return true, start, nil
	}
	// any other number of splits around "-"
	return false, 0, errors.Errorf("invalid value (%v)", val)
}

type keySetBuilder struct {
	out     strings.Builder
	started bool
	comma   bool
	start   int64
	last    int64
}

func (ksb *keySetBuilder) emit() {
	// do we need a separator?
	if !ksb.comma {
		ksb.comma = true
	} else {
		_, _ = ksb.out.WriteString(",")
	}
	// always emit "$START"
	_, _ = ksb.out.WriteString(strconv.FormatInt(ksb.start, 10))
	if ksb.last != ksb.start {
		// for ranges, emit "-$PREV"
		_, _ = ksb.out.WriteString("-")
		_, _ = ksb.out.WriteString(strconv.FormatInt(ksb.last, 10))
	}
}

func (ksb *keySetBuilder) Add(n int64) {
	if !ksb.started {
		// first call to Add()
		ksb.started = true
		ksb.start = n
		ksb.last = n
		return
	}
	if n == ksb.last+1 {
		// extend the current range
		ksb.last++
		return
	}
	// emit previous number or range
	ksb.emit()
	// start new range
	ksb.start = n
	ksb.last = n
}

func (ksb *keySetBuilder) Finish() string {
	if !ksb.started {
		return ""
	}
	// emit final value
	ksb.emit()
	return ksb.out.String()
}

// ProcessKnown takes what the client reports as known keys, combined with which keys the server
// knows exist, and returns what the client should be told is deleted and which keys the client is
// not yet aware of, which should be hydrated by querying the database.
//
// Parameters:
// - `known` is a range-encoded string, directly from the REST API.
// - `exist` is a list of ints, likely from the database or some in-memory cache.
//
// Return Values:
// - `gone` is a range-encoded string, suitable for returning over the REST API.
// - `new` is a list of ints, which are PKs that will need hydrating from the database.
func ProcessKnown(known string, exist []int64) (string, []int64, error) {
	ksi := newKeySetIter(known)

	xIdx := -1
	existNext := func() (bool, int64) {
		xIdx++
		if xIdx >= len(exist) {
			return false, 0
		}
		return true, exist[xIdx]
	}

	var removed keySetBuilder
	var added []int64

	kok, k, err := ksi.Next()
	xok, x := existNext()
	for kok && xok && err == nil {
		switch {
		case k == x:
			// ignore matched values; advance x and k
			kok, k, err = ksi.Next()
			xok, x = existNext()
		case k < x:
			// x is ahead, k must have been removed
			removed.Add(k)
			kok, k, err = ksi.Next()
		default:
			// k is ahead, x must have been added
			added = append(added, x)
			xok, x = existNext()
		}
	}
	for kok && err == nil {
		// if there are extra known values after exist values are exhausted, they are removed
		removed.Add(k)
		kok, k, err = ksi.Next()
	}
	for xok && err == nil {
		// if there are extra exist values after known values are exhausted, they are added
		added = append(added, x)
		xok, x = existNext()
	}
	if err != nil {
		return "", nil, err
	}
	return removed.Finish(), added, nil
}
