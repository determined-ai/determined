package model

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
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

func TestInvalidation(t *testing.T) {
	tt := time.Date(2023, 2, 2, 15, 4, 5, 0, time.UTC)
	extSess := ExternalSessions{
		Invalidations: &InvalidationMap{
			DefaultTime: tt.Add(3 * time.Hour),
			LastUpdated: tt.Add(9 * time.Hour),
			InvalidationTimes: map[string]map[string]time.Time{
				"user-1": {
					"logout":   tt.Add(9 * time.Hour),
					"new-perm": tt.Add(5 * time.Hour),
				},
			},
		},
	}

	j := JWT{
		StandardClaims: jwt.StandardClaims{ //nolint
			IssuedAt:  tt.Unix(),
			ExpiresAt: tt.Add(20 * time.Hour).Unix(),
		},
		UserID:   "user-1",
		Email:    "test@test-dummy.com",
		Name:     "Test User",
		OrgRoles: map[OrgID]OrgRoleClaims{},
	}
	require.ErrorIs(t, extSess.Validate(&j), jwt.ErrTokenExpired)

	j.StandardClaims.IssuedAt = tt.Add(10 * time.Hour).Unix()
	require.NoError(t, extSess.Validate(&j))

	j.StandardClaims.IssuedAt = tt.Add(6 * time.Hour).Unix()
	require.ErrorIs(t, extSess.Validate(&j), jwt.ErrTokenExpired)

	// No such user, using default time
	j.UserID = "user-23"
	require.NoError(t, extSess.Validate(&j))
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
