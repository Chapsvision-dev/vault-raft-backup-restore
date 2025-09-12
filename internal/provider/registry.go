package provider

import "fmt"

// Factory creates a provider instance from opaque config (provider-specific).
type Factory func(any) (Provider, error)

var registry = map[string]Factory{}

// Register binds a provider name to its factory.
func Register(name string, f Factory) {
	registry[name] = f
}

// New returns a provider instance by name.
func New(name string, cfg any) (Provider, error) {
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return f(cfg)
}
