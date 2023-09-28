package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUserNilLastLoginProto(t *testing.T) {
	u := User{}
	require.Nil(t, u.Proto().LastAuthAt)
}

func TestUserNonNilLastLoginProto(t *testing.T) {
	expectedTime := time.Now()
	u := User{LastAuthAt: &expectedTime}
	require.WithinDuration(t, expectedTime, u.Proto().LastAuthAt.AsTime(), time.Millisecond)
}
