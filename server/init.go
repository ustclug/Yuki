package server

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

var (
	IsTest bool
)

func init() {
	if strings.HasPrefix(os.Getenv("YUKI_ENV"), "test") {
		IsTest = true
	}

	viper.SetEnvPrefix("YUKI")
	viper.SetConfigFile("/etc/yuki/daemon.toml")
}
