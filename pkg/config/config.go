package config

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config stores application configuration in a viper instance.
type Config struct {
	*viper.Viper
}

// New returns a new instance of Config, bounded to the provided flagset.
func New(flags *pflag.FlagSet) (*Config, error) {
	v := viper.New()
	if flags != nil {
		if err := v.BindPFlags(flags); err != nil {
			return nil, err
		}
	}

	return &Config{v}, nil
}
