package config

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config stores application configuration in a viper instance.
type Config struct {
	*viper.Viper
}

// New returns a new instance of Config, bounded to the provided config file
// and optional flagset.
func New(configFile string, flags *pflag.FlagSet, configDir ...string) (*Config, error) {
	v := viper.New()

	v.SetConfigName(configFile)
	v.AddConfigPath(".")
	for _, dir := range configDir {
		v.AddConfigPath(dir)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	if flags != nil {
		if err := v.BindPFlags(flags); err != nil {
			return nil, err
		}
	}

	return &Config{v}, nil
}
