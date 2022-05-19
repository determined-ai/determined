package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"golang.org/x/exp/maps"
)

var knownAuthZTypes map[string]bool
var authZConfigMutex sync.Mutex

const BasicAuthZType = "basic"

type AuthZConfig struct {
	Type         string  `json:"type"`
	FallbackType *string `json:"fallback"`
}

func DefaultAuthZConfig() *AuthZConfig {
	return &AuthZConfig{
		Type: BasicAuthZType,
		// TODO(ilia): Maybe default to nil?
		FallbackType: ptrs.Ptr(BasicAuthZType),
	}
}

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

func initAuthZTypes() {
	authZConfigMutex.Lock()
	defer authZConfigMutex.Unlock()

	if knownAuthZTypes != nil {
		return
	}

	knownAuthZTypes = make(map[string]bool)
	knownAuthZTypes[BasicAuthZType] = true
}

func RegisterAuthZType(authzType string) {
	initAuthZTypes()

	authZConfigMutex.Lock()
	defer authZConfigMutex.Unlock()

	knownAuthZTypes[authzType] = true
}

func GetAuthZConfig() AuthZConfig {
	return GetMasterConfig().Security.AuthZ
}
