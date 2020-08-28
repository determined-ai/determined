package api

// effectiveOffset returns effective offset.
func EffectiveOffset(reqOffset int, total int) (offset int) {
	switch {
	case reqOffset < -total:
		return 0
	case reqOffset < 0:
		return total + reqOffset
	default:
		return reqOffset
	}
}

// effectiveLimit returns effective limit.
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

func EffectiveOffsetNLimit(reqOffset int, reqLimit int, totalItems int) (offset int, limit int) {
	offset = EffectiveOffset(reqOffset, totalItems)
	limit = EffectiveLimit(reqLimit, offset, totalItems)
	return offset, limit
}
