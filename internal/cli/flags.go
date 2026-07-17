package cli

import (
	"context"

	"github.com/asphaltbuffet/go-outside/internal/config"
)

// GetConfig retrieves the Config from the context.
func GetConfig(ctx context.Context) (*config.Config, bool) {
	v := ctx.Value(config.ConfigKey)
	cfg, ok := v.(*config.Config)
	return cfg, ok
}

// MustGetConfig retrieves the Config from the context or panics if not found.
func MustGetConfig(ctx context.Context) *config.Config {
	cfg, ok := GetConfig(ctx)
	if !ok {
		panic("go-outside: Config not found in context -- was PersistentPreRunE bypassed?")
	}
	return cfg
}
