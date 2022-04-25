package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type GameServerConfig struct {
	DebugMode      bool
	Port           string
	TickIntervalMS int
}

func LoadGameServerConfig() (sc *GameServerConfig, err error) {
	viper.GetViper().AddConfigPath("config/")
	viper.SetConfigName("game")
	viper.SetConfigType("yaml")

	if err = viper.ReadInConfig(); err != nil {
		err = fmt.Errorf("SGS: %w", err)
		return
	}

	sc = &GameServerConfig{
		Port:           viper.GetString("server.port"),
		DebugMode:      viper.GetBool("server.debugMode"),
		TickIntervalMS: viper.GetInt("server.tickIntervalMS"),
	}

	return
}

type MatchmakingServerConfig struct {
	DebugMode bool
	Port      string
}

func LoadMatchmakingServerConfig() (sc *MatchmakingServerConfig, err error) {
	viper.GetViper().AddConfigPath("config/")
	viper.SetConfigName("matchmaking")
	viper.SetConfigType("yaml")

	if err = viper.ReadInConfig(); err != nil {
		err = fmt.Errorf("SGS: %w", err)
		return
	}

	sc = &MatchmakingServerConfig{
		Port:      viper.GetString("server.port"),
		DebugMode: viper.GetBool("server.debugMode"),
	}

	return
}
