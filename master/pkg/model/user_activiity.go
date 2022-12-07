package model

import (
	"fmt"
	"time"

	"github.com/determined-ai/determined/proto/pkg/userv1"
	"github.com/uptrace/bun"
)

// ActivityType describes a user activity.
type ActivityType string

// EntityType represents an entity.
type EntityType string

const (
	// ActivityTypeGet represents a get request.
	ActivityTypeGet ActivityType = "GET"
)

const (
	// EntityTypeProject represents a project.
	EntityTypeProject EntityType = "Project"
)

// UserActivity is a record of user activity.
type UserActivity struct {
	bun.BaseModel `bun:"table:activity"`
	UserID        UserID       `db:"user_id" json:"user_id"`
	ActivityType  ActivityType `db:"activity_type" json:"activity_type"`
	EntityType    EntityType   `db:"entity_type" json:"entity_type"`
	EntityID      int32        `db:"entity_id" json:"entity_id"`
	ActivityTime  time.Time    `db:"activity_time" json:"activity_time"`
}

// EntityTypeFromProto returns an EntityType from a proto.
func entityTypeFromProto(e userv1.EntityType) EntityType {
	switch e {
	case userv1.EntityType_ENTITY_TYPE_PROJECT:
		return EntityTypeProject
	default:
		panic(fmt.Errorf("missing mapping for entity type %s to model", e))
	}
}

// ActivityTypeFromProto returns an ActivityType from a proto.
func activityTypeFromProto(a userv1.ActivityType) ActivityType {
	switch a {
	case userv1.ActivityType_ACTIVITY_TYPE_GET:
		return ActivityTypeGet
	default:
		panic(fmt.Errorf("missing mapping for activity type %s to model", a))
	}
}

// UserActivityFromProto returns a model UserActivity from a proto definition.
func UserActivityFromProto(
	a userv1.ActivityType,
	e userv1.EntityType,
	entityID int32,
	userID int32,
	timestamp time.Time,
) *UserActivity {
	return &UserActivity{
		UserID:       UserID(userID),
		ActivityType: activityTypeFromProto(a),
		EntityType:   entityTypeFromProto(e),
		EntityID:     entityID,
		ActivityTime: timestamp,
	}
}
