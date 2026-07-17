package config

import (
	"fmt"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

func loadConfig(fs afero.Fs, fp string) (*Config, error) {
	v := viper.New()
	v.SetFs(fs)
	v.SetEnvPrefix("GO_OUTSIDE")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetConfigType("toml")

	var cfg Config

	if fp != "" {
		v.SetConfigFile(fp)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("reading config %q: %w", fp, err)
		}
	} else {
		// Check env var first
		if envPath := v.GetString("config"); envPath != "" {
			v.SetConfigFile(envPath)
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("reading config from env: %w", err)
			}
		} else {
			v.SetConfigName("go-outside")
			v.AddConfigPath(DefaultConfigPath())
			_ = v.ReadInConfig() // missing config file is not an error
		}
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}
