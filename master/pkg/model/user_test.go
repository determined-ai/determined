package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUserNilLastLoginProto(t *testing.T) {
	u := User{}
	require.Nil(t, u.Proto().LastLogin)
}

func TestUserNonNilLastLoginProto(t *testing.T) {
	expectedTime := time.Now()
	u := User{LastLogin: &expectedTime}
	require.WithinDuration(t, expectedTime, u.Proto().LastLogin.AsTime(), time.Millisecond)
}
