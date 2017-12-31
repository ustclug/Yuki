package server

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

var (
	IsTest bool = false
)

func init() {
	if strings.HasPrefix(os.Getenv("YUKI_ENV"), "test") {
		IsTest = true
	}

	viper.SetEnvPrefix("YUKI")
	viper.SetConfigFile("/etc/yuki/daemon.toml")

	viper.SetDefault("debug", false)
	viper.SetDefault("db_name", "mirror")
	viper.SetDefault("docker_endpoint", "unix:///var/run/docker.sock")

	viper.SetDefault("owner", "0:0")
	viper.SetDefault("log_dir", "/var/log/yuki/")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("name_prefix", "syncing-")
	viper.SetDefault("sync_timeout", "48h")
	viper.SetDefault("allow_origins", []string{"*"})
}
