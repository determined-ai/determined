package stream

import (
	"strings"
	"strconv"

	"github.com/pkg/errors"
	// log "github.com/sirupsen/logrus"
)

// func decodeKnown(known string) ([]int64, error) {
// 	vals := strings.Split(known, ",")
// 	var out []int64
// 	for _, v := range vals {
// 		if strings.Contains(v, "-") {
// 			// a range
// 			subs := strings.Split(v, "-")
// 			if len(subs) != 2 {
// 				return nil, errors.Errorf("invalid range (%v)", v)
// 			}
// 			start, err := strconv.ParseInt(subs[0], 10, 64)
// 			if err != nil {
// 				return nil, errors.Wrapf(err, "invalid range (%v)", v)
// 			}
// 			end, err := strconv.ParseInt(subs[1], 10, 64)
// 			if err != nil {
// 				return nil, errors.Wrapf(err, "invalid range (%v)", v)
// 			}
// 			if start > end {
// 				start, end = end, start
// 			}
// 			// emit every value in the range (including endpoints)
// 			for i := start; i <= end; i++ {
// 				out = append(out, i)
// 			}
// 		} else {
// 			// a single number
// 			n, err := strconv.ParseInt(v, 10, 64)
// 			if err != nil {
// 				return nil, errors.Wrapf(err, "invalid number (%v)", v)
// 			}
// 			out = append(out, n)
// 		}
// 	}
// 	return out, nil
// }
//
// func encodeDeleted(known []int64) string {
// 	if len(known) == 0 {
// 		return ""
// 	}
// 	var out strings.Builder
// 	start := known[0]
// 	last := known[0]
// 	comma := false
//
// 	emit := func() {
// 		// do we need a separator?
// 		if !comma {
// 			comma = true
// 		} else {
// 			_, _ = out.WriteString(",")
// 		}
// 		// always emit "$START"
// 		_, _ = out.WriteString(strconv.FormatInt(start, 10))
// 		if last != start {
// 			// for ranges, emit "-$PREV"
// 			_, _ = out.WriteString("-")
// 			_, _ = out.WriteString(strconv.FormatInt(last, 10))
// 		}
// 	}
//
// 	for _, k := range known[1:] {
// 		if k == last + 1 {
// 			// extend the current range
// 			last++
// 		} else {
// 			// emit previous number or range
// 			emit()
// 			// start new range
// 			start = k
// 			last = k
// 		}
// 	}
// 	// emit final value
// 	emit()
// 	return out.String()
// }

type KeySetIter struct {
	vals []string
	idx int
	start int64
	end int64
}

func NewKeySetIter(keyset string) *KeySetIter {
	return &KeySetIter{vals: strings.Split(keyset, ",")}
}

// Next returns a tuple of (ok, value, err).
func (ksi *KeySetIter) Next() (bool, int64, error) {
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
		ksi.end = end + 1  // plus one to save an exclusive endpoint
		return true, start, nil
	}
	// any other number of splits around "-"
	return false, 0, errors.Errorf("invalid value (%v)", val)
}

type KeySetBuilder struct {
	out strings.Builder
	started bool
	comma bool
	start int64
	last int64
}

func (ksb *KeySetBuilder) emit() {
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

func (ksb *KeySetBuilder) Add(n int64) {
	if !ksb.started {
		// first call to Add()
		ksb.started = true
		ksb.start = n
		ksb.last = n
		return
	}
	if n == ksb.last + 1 {
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

func (ksb *KeySetBuilder) Finish() string {
	// emit final value
	ksb.emit()
	return ksb.out.String()
}

// processKnown takes what the client reports as known keys, combined with which keys the server
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
func processKnown(known string, exist []int64) (string, []int64, error) {
	ksi := NewKeySetIter(known)

	xIdx := -1
	existNext := func() (bool, int64) {
		xIdx++
		if xIdx >= len(exist) {
			return false, 0
		}
		return true, exist[xIdx]
	}

	var removed KeySetBuilder
	var added []int64

	kok, k, err := ksi.Next()
	xok, x := existNext()
	for kok && xok && err == nil {
		if k == x {
			// ignore matched values; advance x and k
			kok, k, err = ksi.Next()
			xok, x = existNext()
		} else if k < x {
			// x is ahead, k must have been removed
			removed.Add(k)
			kok, k, err = ksi.Next()
		} else {
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
