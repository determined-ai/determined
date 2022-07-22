package authz

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/config"
)

// AuthZProviderType is a per-module registry for authz implementations.
type AuthZProviderType[T any] struct {
	registry map[string]T
	once     sync.Once
}

// Register adds new implementation.
func (p *AuthZProviderType[T]) Register(authZType string, impl T) {
	p.once.Do(func() {
		p.registry = make(map[string]T)
	})
	// TODO(ilia): keep registry here or global in internal/authz/authz_basic.go
	config.RegisterAuthZType(authZType)
	if _, ok := p.registry[authZType]; ok {
		panic(fmt.Errorf("can't do double register of type %s for: %s", authZType, p.string()))
	}
	p.registry[authZType] = impl
}

func (p *AuthZProviderType[T]) string() string {
	return reflect.TypeOf((*T)(nil)).Elem().String()
}

// Get returns the selected implementation.
func (p *AuthZProviderType[T]) Get() T {
	if len(p.registry) == 0 {
		panic(fmt.Errorf("empty registry for: %s", p.string()))
	}

	authZConfig := config.GetAuthZConfig()

	res, ok := p.registry[authZConfig.Type]
	if ok {
		return res
	}

	okTypes := strings.Join(maps.Keys(p.registry), ", ")

	if authZConfig.FallbackType == nil {
		panic(fmt.Errorf(
			"failed to find authz type %s in %s for: %s",
			authZConfig.Type, okTypes, p.string()))
	}

	res, ok = p.registry[*authZConfig.FallbackType]
	if ok {
		return res
	}

	panic(fmt.Errorf(
		"failed to find authz types %s, %s in %s for: %s",
		authZConfig.Type, *authZConfig.FallbackType, okTypes, p.string()))
}
