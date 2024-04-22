package model

import (
	"github.com/golang-jwt/jwt/v4"
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
	UserID   string // SaaS user IDs are strings, unlike Determined's int-based type
	Email    string
	Name     string
	OrgRoles map[OrgID]OrgRoleClaims
}

// SCIMEmailsFromJWT returns a consistent SCIMEmails struct wrapping the single email in a JWT.
func SCIMEmailsFromJWT(claims *JWT) SCIMEmails {
	return SCIMEmails{
		SCIMEmail{
			Type:    "OpenID Connect",
			SValue:  claims.Email,
			Primary: true,
		},
	}
}

// SCIMNameFromJWT returns a consistent SCIMName struct wrapping the single name in a JWT.
func SCIMNameFromJWT(claims *JWT) SCIMName {
	return SCIMName{
		// TODO consider getting separate names from OpenID profile
		GivenName: claims.Name,
	}
}
