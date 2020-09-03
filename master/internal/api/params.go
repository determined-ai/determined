package api

import "math"

// EffectiveOffset returns effective offset.
func EffectiveOffset(reqOffset int, total int) (offset int) {
	switch {
	case reqOffset < -total:
		return 0
	case reqOffset < 0:
		return int(math.Max(float64(total+reqOffset), 0))
	default:
		return reqOffset
	}
}

// EffectiveLimit returns effective limit.
// Input: non-negative offset and limit.
func EffectiveLimit(limit int, offset int, total int) int {
	switch {
	case limit <= 0:
		return -1
	case limit > total-offset:
		return total - offset
	default:
		return limit
	}
}

// EffectiveOffsetNLimit calculates effective offset and limit.
func EffectiveOffsetNLimit(reqOffset int, reqLimit int, totalItems int) (offset int, limit int) {
	offset = EffectiveOffset(reqOffset, totalItems)
	limit = EffectiveLimit(reqLimit, offset, totalItems)
	return offset, limit
}
