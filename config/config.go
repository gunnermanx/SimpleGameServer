package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	DebugMode bool
	Port      string
}

func LoadServerConfig() (sc *ServerConfig, err error) {
	viper.GetViper().AddConfigPath("config/")
	viper.SetConfigName("server")
	viper.SetConfigType("yaml")

	if err = viper.ReadInConfig(); err != nil {
		err = fmt.Errorf("SGS: %w", err)
		return
	}

	sc = &ServerConfig{
		Port:      viper.GetString("server.port"),
		DebugMode: viper.GetBool("server.debugMode"),
	}

	return
}
