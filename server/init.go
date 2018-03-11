package server

import (
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("YUKI")
	viper.SetConfigFile("/etc/yuki/daemon.toml")
}
