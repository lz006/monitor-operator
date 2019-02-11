package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func LoadConfig() {
	// Load config
	viper.AddConfigPath("/etc/monitor-operator")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	viper.SetConfigName("monitor-operator-conf")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
}
