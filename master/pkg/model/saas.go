package model

import (
	"github.com/golang-jwt/jwt"
)

// This should all correspond to models in the SaaS code

// ClusterID is a string intended specifically as a cluster ID.
type ClusterID string

// OrgID is a string intended specifically as a organization ID.
type OrgID string

// Role is a string intended specifically as an access level.
type Role string

const (
	// NoRole implies previous access has been revoked.
	NoRole Role = "none"

	// UserRole implies normal worker access.
	UserRole Role = "user"

	// AdminRole implies management / administrative access.
	AdminRole Role = "admin"
)

// OrgRoleClaims is the specification of all permissions a user has in a given org.
type OrgRoleClaims struct {
	Role               Role
	DefaultClusterRole Role
	ClusterRoles       map[ClusterID]Role
}

// JWT defines the claims that are serialized and signed to make a bearer token.
type JWT struct {
	jwt.StandardClaims
	Subject  string
	Email    string
	OrgRoles map[OrgID]OrgRoleClaims
}
