package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUserNilLastAuthAtProto(t *testing.T) {
	u := User{}
	require.Nil(t, u.Proto().LastAuthAt)
}

func TestUserNonNilLastAuthAtProto(t *testing.T) {
	expectedTime := time.Now()
	u := User{LastAuthAt: &expectedTime}
	require.WithinDuration(t, expectedTime, u.Proto().LastAuthAt.AsTime(), time.Millisecond)
}

func TestUserWebSetting_Proto(t *testing.T) {
	in := UserWebSetting{
		UserID:      UserID(42),
		Key:         uuid.NewString(),
		Value:       uuid.NewString(),
		StoragePath: uuid.NewString(),
	}
	out := in.Proto()
	require.Equal(t, in.Key, out.Key)
	require.Equal(t, in.Value, out.Value)
	require.Equal(t, in.StoragePath, out.StoragePath)
}
