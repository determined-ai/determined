package errors

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrorTimeoutRetry(t *testing.T) {
	testErr := errors.New("test error")
	errInfo := NewErrorTimeoutRetry(30*time.Second, 3)
	assert.Equal(t, errInfo.GetError(), nil)

	for i := 0; i < 3; i++ {
		errInfo.SetError(fmt.Errorf("tmp error %d", i))
		assert.Equal(t, errInfo.GetError(), nil)
	}

	errInfo.SetError(testErr)
	assert.Equal(t, errInfo.GetError(), testErr)

	errInfo.SetError(nil)

	for i := 0; i < 3; i++ {
		errInfo.SetError(fmt.Errorf("tmp after set error %d", i))
		assert.Equal(t, errInfo.GetError(), nil)
	}

	errInfo.SetError(testErr)
	assert.Equal(t, errInfo.GetError(), testErr)

	errInfo.time = time.Now().Add(time.Second)
	assert.Equal(t, errInfo.GetError(), testErr)

	errInfo.time = time.Now().Add(-15 * time.Second)
	assert.Equal(t, errInfo.GetError(), testErr)

	errInfo.time = time.Now().Add(-29 * time.Second)
	assert.Equal(t, errInfo.GetError(), testErr)

	errInfo.time = time.Now().Add(-31 * time.Second)
	assert.Equal(t, errInfo.GetError(), nil)

	errInfo.time = time.Now().Add(-48 * time.Hour)
	assert.Equal(t, errInfo.GetError(), nil)
}

func TestErrorNoTimeoutNoRetry(t *testing.T) {
	errInfo := NewErrorTimeoutRetry(0, 0)
	assert.Equal(t, errInfo.GetError(), nil)

	for i := 0; i < 100; i++ {
		errInfo.SetError(fmt.Errorf("tmp error %d", i))
		assert.Equal(t, errInfo.GetError(), nil)
	}

	errInfo.time = time.Now().Add(-60 * time.Second)
	assert.Equal(t, errInfo.GetError(), nil)

	errInfo.time = time.Now().Add(-48 * time.Hour)
	assert.Equal(t, errInfo.GetError(), nil)
}
