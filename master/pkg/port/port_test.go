package port

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRange(t *testing.T) {
	tests := []struct {
		name      string
		start     int
		end       int
		usedPorts []int
		expectErr bool
	}{
		{"Valid range", 1000, 2000, []int{1500, 1600}, false},
		{"Invalid range, start greater than end", 2000, 1000, []int{}, true},
		{"Invalid range, negative start", -1000, 2000, []int{}, true},
		{"Invalid range, end greater than 65535", 1000, 70000, []int{}, true},
		{"Used port out of range", 1000, 2000, []int{3000}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRange(tt.start, tt.end, tt.usedPorts)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRange_LoadInUsedPorts(t *testing.T) {
	r, err := NewRange(1000, 2000, []int{})
	require.NoError(t, err)

	err = r.LoadInUsedPorts([]int{1500, 1600})
	assert.NoError(t, err)

	err = r.LoadInUsedPorts([]int{3000})
	assert.Error(t, err)

	err = r.LoadInUsedPorts([]int{1100, 1900})
	assert.NoError(t, err)
}

func TestRange_nextAvailablePort(t *testing.T) {
	r, err := NewRange(1000, 1002, []int{1000})
	require.NoError(t, err)

	port, err := r.nextAvailablePort()
	assert.NoError(t, err)
	assert.Equal(t, 1001, port)

	port, err = r.nextAvailablePort()
	assert.NoError(t, err)
	assert.Equal(t, 1002, port)

	_, err = r.nextAvailablePort()
	assert.Error(t, err)
}

func TestRange_GetAndMarkUsed(t *testing.T) {
	r, err := NewRange(1000, 1004, []int{1000})
	require.NoError(t, err)

	ports, err := r.GetAndMarkUsed(2)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []int{1001, 1002}, ports)

	_, err = r.GetAndMarkUsed(3)
	assert.Error(t, err)

	r, err = NewRange(1000, 1004, []int{1000, 1001, 1002})
	require.NoError(t, err)

	ports, err = r.GetAndMarkUsed(2)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []int{1003, 1004}, ports)

	_, err = r.GetAndMarkUsed(1)
	assert.Error(t, err)
}

func TestRange_MarkPortAsUsed(t *testing.T) {
	r, err := NewRange(1000, 2000, []int{})
	require.NoError(t, err)

	err = r.MarkPortAsUsed(1500)
	assert.NoError(t, err)

	err = r.MarkPortAsUsed(1500)
	assert.Error(t, err)

	err = r.MarkPortAsUsed(3000)
	assert.Error(t, err)

	err = r.MarkPortAsUsed(2000)
	assert.NoError(t, err)
}

func TestRange_MarkPortAsFree(t *testing.T) {
	r, err := NewRange(1000, 2000, []int{1500})
	require.NoError(t, err)

	err = r.MarkPortAsFree(1500)
	assert.NoError(t, err)

	err = r.MarkPortAsFree(1500)
	assert.Error(t, err)

	err = r.MarkPortAsUsed(1600)
	assert.NoError(t, err)
	err = r.MarkPortAsFree(1600)
	assert.NoError(t, err)
	err = r.MarkPortAsFree(1600)
	assert.Error(t, err)
}
