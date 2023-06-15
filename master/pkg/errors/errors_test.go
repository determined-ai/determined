package errors

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrorTimeoutRetry(t *testing.T) {
	testErr := fmt.Errorf("test error")
	errInfo := NewStickyError(30*time.Second, 3)
	assert.Equal(t, errInfo.Error(), nil)

	for i := 0; i < 3; i++ {
		assert.Equal(t, errInfo.SetError(fmt.Errorf("tmp error %d", i)), nil)
	}

	assert.Equal(t, errInfo.SetError(testErr), testErr)

	_ = errInfo.SetError(nil)

	for i := 0; i < 3; i++ {
		assert.Equal(t, errInfo.SetError(fmt.Errorf("tmp after set error %d", i)), nil)
	}

	assert.Equal(t, errInfo.SetError(testErr), testErr)

	errInfo.time = time.Now().Add(time.Second)
	assert.Equal(t, errInfo.Error(), testErr)

	errInfo.time = time.Now().Add(-15 * time.Second)
	assert.Equal(t, errInfo.Error(), testErr)

	errInfo.time = time.Now().Add(-29 * time.Second)
	assert.Equal(t, errInfo.Error(), testErr)

	errInfo.time = time.Now().Add(-31 * time.Second)
	assert.Equal(t, errInfo.Error(), nil)

	errInfo.time = time.Now().Add(-48 * time.Hour)
	assert.Equal(t, errInfo.Error(), nil)

	for i := 0; i < 3; i++ {
		assert.Equal(t, errInfo.SetError(fmt.Errorf("tmp after timeout error %d", i)), nil)
	}

	_ = errInfo.SetError(testErr)
	assert.Equal(t, errInfo.Error(), testErr)
}

func TestErrorNoTimeoutNoRetry(t *testing.T) {
	errInfo := NewStickyError(0, 0)
	assert.Equal(t, errInfo.Error(), nil)

	for i := 0; i < 100; i++ {
		assert.Equal(t, errInfo.SetError(fmt.Errorf("tmp error %d", i)), nil)
	}

	errInfo.time = time.Now().Add(-60 * time.Second)
	assert.Equal(t, errInfo.Error(), nil)

	errInfo.time = time.Now().Add(-48 * time.Hour)
	assert.Equal(t, errInfo.Error(), nil)
}
