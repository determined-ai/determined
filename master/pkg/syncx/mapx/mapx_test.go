package mapx

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
)

// FuzzMap tests for bad concurrent access. Run the command below to test it, if you change the map.
//
//	go test -fuzz FuzzMap --parallel 64 github.com/determined-ai/determined/master/pkg/syncx/mapx
func FuzzMap(f *testing.F) {
	m := New[uint8, string]()

	f.Add(uint8(0), uint8(0), "hello")
	f.Fuzz(func(t *testing.T, op, k uint8, v string) {
		switch op % 7 {
		case 0:
			m.Len()
		case 1:
			m.Store(k, v)
		case 2:
			_, _ = m.Load(k)
		case 3:
			m.Delete(k)
		case 4:
			var _ []uint8
			m.WithLock(func(m map[uint8]string) {
				_ = maps.Keys(m)
			})
		case 5:
			_ = m.Values()
		case 6:
			m.Clear()
		}
	})
}

func TestMapx(t *testing.T) {
	testMap := New[string, string]()
	testMap.Store("1234", "hi")
	testMap.Store("1235", "hello")
	testMap.Store("1236", "world")

	assert.Equal(t, 3, testMap.Len())

	value, _ := testMap.Load("1235")
	assert.Equal(t, "hello", value)

	expectedValueList := [...]string{"hi", "world"}
	testMap.Delete("1235")
	valueList := testMap.Values()
	assert.Equal(t, 2, testMap.Len())

	sort.Strings(valueList)
	for i, v := range valueList {
		assert.Equal(t, expectedValueList[i], v)
	}

	testMap.Clear()
	assert.Empty(t, testMap.Len())
}
