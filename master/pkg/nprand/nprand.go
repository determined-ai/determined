// This package reproduces the Python/Numpy random number generator, which itself is based the C
// library RandomKit, which is based on the original Mersenne Twister code, albeit with many
// modifications.

package nprand

import "fmt"

const (
	stateLen  int    = 624
	maxUint32 uint32 = 0xffffffff
	// Magic Mersenne Twister constants
	mtN       int    = 624
	mtM       int    = 397
	matrixA   uint32 = 0x9908b0df
	upperMask uint32 = 0x80000000
	lowerMask uint32 = 0x7fffffff
)

// State is the state of the random number generator.
type State struct {
	Key [stateLen]uint32 `json:"key"`
	Pos int              `json:"pos"`
}

// New creates a new seeded RNG state.
func New(seed uint32) *State {
	state := State{}
	state.Seed(seed)
	return &state
}

// Seed initializes the RNG state.
func (state *State) Seed(seed uint32) {
	for pos := 0; pos < stateLen; pos++ {
		state.Key[pos] = seed
		seed = (uint32(1812433253)*(seed^(seed>>uint32(30))) + uint32(pos) + 1)
	}
	state.Pos = stateLen
}

// Bits32 generates 32 bits of randomness.
func (state *State) Bits32() uint32 {
	var y uint32
	if state.Pos == stateLen {
		i := 0
		for ; i < mtN-mtM; i++ {
			y = (state.Key[i] & upperMask) | (state.Key[i+1] & lowerMask)
			state.Key[i] = state.Key[i+mtM] ^ (y >> 1) ^ (-(y & 1) & matrixA)
		}
		for ; i < mtN-1; i++ {
			y = (state.Key[i] & upperMask) | (state.Key[i+1] & lowerMask)
			state.Key[i] = state.Key[i+(mtM-mtN)] ^ (y >> 1) ^ (-(y & 1) & matrixA)
		}
		y = (state.Key[mtN-1] & upperMask) | (state.Key[0] & lowerMask)
		state.Key[mtN-1] = state.Key[mtM-1] ^ (y >> 1) ^ (-(y & 1) & matrixA)

		state.Pos = 0
	}
	y = state.Key[state.Pos]
	state.Pos++

	// Tempering
	y ^= y >> 11
	y ^= (y << 7) & uint32(0x9d2c5680)
	y ^= (y << 15) & uint32(0xefc60000)
	y ^= y >> 18

	return y
}

// Bits64 generates 64 bits of randomness.
func (state *State) Bits64() uint64 {
	upper := uint64(state.Bits32()) << 32
	lower := uint64(state.Bits32())
	return upper | lower
}

// Read implements the Reader interface, yielding a random stream of bytes.
func (state *State) Read(p []byte) (int, error) {
	pos := 0
	var val uint32
	for n := 0; n < len(p); n++ {
		if pos == 0 {
			val = state.Bits32()
			pos = 4
		}
		p[n] = byte(val)
		val >>= 8
		pos--
	}
	return len(p), nil
}

// bitsLimit is an internal utility function to generate bits of randomness in [0, limit].
func (state *State) bitsLimit(limit uint64) uint64 {
	if limit == 0 {
		return 0
	}

	// The plan is to generate some random bits, zero out bits above the limit using a mask, and
	// repeat until we get at or below the limit.

	// Compute the smallest bit mask >= limit.
	mask := limit
	mask |= mask >> 1
	mask |= mask >> 2
	mask |= mask >> 4
	mask |= mask >> 8
	mask |= mask >> 16
	mask |= mask >> 32

	// If we only need 32 bits or less, only generate 32 bits or randomness.
	if limit <= uint64(maxUint32) {
		for {
			if val := uint64(state.Bits32()) & mask; val <= limit {
				return val
			}
		}
	}
	// Otherwise generate 64 bits.
	for {
		if val := state.Bits64() & mask; val <= limit {
			return val
		}
	}
}

// Int64 generates a random Int64 in [low, high).  It panics if high <= low.
func (state *State) Int64(low, high int64) int64 {
	if high <= low {
		panic(fmt.Sprintf("nprand Int64: high %v <= low %v", high, low))
	}
	return low + int64(state.bitsLimit(uint64(high)-uint64(low)-1))
}

// Int64n generates a random Int64 in [0, n).  It panics if n <= 0.
func (state *State) Int64n(n int64) int64 {
	if n < 0 {
		panic(fmt.Sprintf("nprand Int64n: n %v < 0", n))
	}
	return int64(state.bitsLimit(uint64(n) - 1))
}

// Intn generates a random Int in [0, n).  It panics if n <= 0.
func (state *State) Intn(n int) int {
	if n < 0 {
		panic(fmt.Sprintf("nprand Intn: n %v < 0", n))
	}
	return int(state.bitsLimit(uint64(n) - 1))
}

// UnitInterval generates a random float64 in [0,1).
func (state *State) UnitInterval() float64 {
	// shifts : 67108864 = 0x4000000, 9007199254740992 = 0x20000000000000
	a := float64(state.Bits32() >> 5)
	b := float64(state.Bits32() >> 6)
	return (a*(1<<26) + b) / (1 << 53)
}

// Uniform generates a random float64 uniformly distributed in [low, high).  It panics if high <=
// low.
func (state *State) Uniform(low, high float64) float64 {
	if high <= low {
		panic(fmt.Sprintf("nprand Uniform: high %v <= low %v", high, low))
	}
	return low + (high-low)*state.UnitInterval()
}
