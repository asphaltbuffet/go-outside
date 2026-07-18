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

	if err := selectConfigSource(v, fp); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// selectConfigSource points v at the config file to read: an explicit path, then
// the GO_OUTSIDE_CONFIG env var, then the default XDG location (whose absence is
// not an error). An explicit path or env path that fails to read IS an error.
func selectConfigSource(v *viper.Viper, fp string) error {
	if fp != "" {
		v.SetConfigFile(fp)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("reading config %q: %w", fp, err)
		}
		return nil
	}

	if envPath := v.GetString("config"); envPath != "" {
		v.SetConfigFile(envPath)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("reading config from env: %w", err)
		}
		return nil
	}

	v.SetConfigName("go-outside")
	v.AddConfigPath(DefaultConfigPath())
	_ = v.ReadInConfig() // missing config file is not an error
	return nil
}
