package main

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
)

var _config *config

type config struct {
	// application config
	*viper.Viper

	// k8s client config
	k8s *rest.Config
}

func initConfig(k8sConfig *rest.Config, flags *pflag.FlagSet) error {
	v := viper.New()
	if err := v.BindPFlags(flags); err != nil {
		return err
	}

	_config = &config{v, k8sConfig}
	return nil
}
