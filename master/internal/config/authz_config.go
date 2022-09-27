package config

import (
	"fmt"
	"strings"
	"sync"

	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

var (
	knownAuthZTypes  map[string]bool
	authZConfigMutex sync.Mutex
)

// BasicAuthZType is the default authz string id.
const BasicAuthZType = "basic"

// AuthZConfig is a authz-related section of master config.
type AuthZConfig struct {
	Type          string  `json:"type"`
	FallbackType  *string `json:"fallback"`
	RBACUIEnabled *bool   `json:"rbac_ui_enabled"`
	// Removed: this option is removed and will not have any effect.
	StrictNTSCEnabled      bool                         `json:"_strict_ntsc_enabled"`
	AssignWorkspaceCreator AssignWorkspaceCreatorConfig `json:"workspace_creator_assign_role"`
}

// DefaultAuthZConfig returns default authz config.
func DefaultAuthZConfig() *AuthZConfig {
	return &AuthZConfig{
		Type: BasicAuthZType,
		// TODO(ilia): Maybe default to nil?
		FallbackType: ptrs.Ptr(BasicAuthZType),
		AssignWorkspaceCreator: AssignWorkspaceCreatorConfig{
			Enabled: true,
			RoleID:  2, // WorkspaceAdmin.
		},
	}
}

// Validate the authz config.
func (c *AuthZConfig) Validate() []error {
	var errs []error

	okTypes := strings.Join(maps.Keys(knownAuthZTypes), ", ")
	errorTmpl := "\"%s\" is not a known authz type, must be one of: %s"
	if _, ok := knownAuthZTypes[c.Type]; !ok {
		errs = append(errs, fmt.Errorf(errorTmpl, c.Type, okTypes))
	}

	if c.FallbackType != nil {
		if _, ok := knownAuthZTypes[*c.FallbackType]; !ok {
			errs = append(errs, fmt.Errorf(errorTmpl, *c.FallbackType, okTypes))
		}
	}

	return errs
}

// AssignWorkspaceCreatorConfig configures behavior of assigning a role on workspace creation.
type AssignWorkspaceCreatorConfig struct {
	Enabled bool `json:"enabled"`
	RoleID  int  `json:"role_id"`
}

// Validate the RoleID of the config.
func (a AssignWorkspaceCreatorConfig) Validate() []error {
	if a.RoleID <= 0 {
		return []error{
			fmt.Errorf("workspace_creator_assign_role.role_id must be >= 0 got %d", a.RoleID),
		}
	}
	return nil
}

// IsRBACUIEnabled returns if the feature flag RBAC should be enabled.
func (c AuthZConfig) IsRBACUIEnabled() bool {
	if c.RBACUIEnabled != nil {
		return *c.RBACUIEnabled
	}
	return c.Type != BasicAuthZType
}

func initAuthZTypes() {
	authZConfigMutex.Lock()
	defer authZConfigMutex.Unlock()

	if knownAuthZTypes != nil {
		return
	}

	knownAuthZTypes = make(map[string]bool)
	knownAuthZTypes[BasicAuthZType] = true
}

// RegisterAuthZType adds new known authz type.
func RegisterAuthZType(authzType string) {
	initAuthZTypes()

	authZConfigMutex.Lock()
	defer authZConfigMutex.Unlock()

	knownAuthZTypes[authzType] = true
}

// GetAuthZConfig returns current global authz config.
func GetAuthZConfig() AuthZConfig {
	return GetMasterConfig().Security.AuthZ
}
