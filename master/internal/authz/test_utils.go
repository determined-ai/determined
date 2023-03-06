//go:build integration
// +build integration

package authz

// RegisterOverride adds new implementation overwriting any existing one.
func (p *AuthZProviderType[T]) RegisterOverride(authZType string, impl T) {
	p.once.Do(func() {
		p.registry = make(map[string]T)
	})
	delete(p.registry, authZType)
	p.Register(authZType, impl)
}
