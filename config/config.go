package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port string
}

func LoadServerConfig() (sc ServerConfig, err error) {
	viper.GetViper().AddConfigPath("config/")
	viper.SetConfigName("server")
	viper.SetConfigType("yaml")

	if err = viper.ReadInConfig(); err != nil {
		err = fmt.Errorf("failed to read in server config: %w", err)
		return
	}

	sc = ServerConfig{
		Port: viper.GetString("server.port"),
	}

	return
}
