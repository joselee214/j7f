package config

import (
	"github.com/spf13/viper"
)

type Configer struct {
	*viper.Viper
}

func NewConfig() (*Configer, error) {
	v := NewViper()
	return v, nil
}
