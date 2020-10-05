package api

// EffectiveOffset translates negative offsets into positive ones.
func EffectiveOffset(offset, total int) int {
	switch {
	case offset < -total:
		return 0
	case offset < 0:
		return max(total+offset, 0)
	default:
		return offset
	}
}

// EffectiveLimit computes a hard limit based on the offset and total available items if there is a
// limit set.
// Input: non-negative offset
func EffectiveLimit(limit, offset, total int) int {
	if offset < 0 {
		panic("input offset has to be non-negative")
	}
	switch {
	case limit <= 0:
		return -1
	case limit > total-offset:
		return total - offset
	default:
		return limit
	}
}

// EffectiveOffsetAndLimit calculates effective offset and limit.
func EffectiveOffsetAndLimit(offset, limit int, totalItems int) (int, int) {
	eOffset := EffectiveOffset(offset, totalItems)
	eLimit := EffectiveLimit(limit, eOffset, totalItems)
	return eOffset, eLimit
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
