package errors

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorTimeoutRetry(t *testing.T) {
	testErr := fmt.Errorf("test error")
	errInfo := NewStickyError(30*time.Second, 3)
	require.NoError(t, errInfo.Error())

	for i := 0; i < 3; i++ {
		require.NoError(t, errInfo.SetError(fmt.Errorf("tmp error %d", i)))
	}

	assert.Equal(t, errInfo.SetError(testErr), testErr)

	_ = errInfo.SetError(nil)

	for i := 0; i < 3; i++ {
		require.NoError(t, errInfo.SetError(fmt.Errorf("tmp after set error %d", i)))
	}

	assert.Equal(t, errInfo.SetError(testErr), testErr)

	errInfo.time = time.Now().Add(time.Second)
	assert.Equal(t, errInfo.Error(), testErr)

	errInfo.time = time.Now().Add(-15 * time.Second)
	assert.Equal(t, errInfo.Error(), testErr)

	errInfo.time = time.Now().Add(-29 * time.Second)
	assert.Equal(t, errInfo.Error(), testErr)

	errInfo.time = time.Now().Add(-31 * time.Second)
	require.NoError(t, errInfo.Error())

	errInfo.time = time.Now().Add(-48 * time.Hour)
	require.NoError(t, errInfo.Error())

	for i := 0; i < 3; i++ {
		require.NoError(t, errInfo.SetError(fmt.Errorf("tmp after timeout error %d", i)))
	}

	_ = errInfo.SetError(testErr)
	assert.Equal(t, errInfo.Error(), testErr)
}

func TestErrorNoTimeoutNoRetry(t *testing.T) {
	errInfo := NewStickyError(0, 0)
	require.NoError(t, errInfo.Error())

	for i := 0; i < 100; i++ {
		require.NoError(t, errInfo.SetError(fmt.Errorf("tmp error %d", i)))
	}

	errInfo.time = time.Now().Add(-60 * time.Second)
	require.NoError(t, errInfo.Error())

	errInfo.time = time.Now().Add(-48 * time.Hour)
	require.NoError(t, errInfo.Error())
}
