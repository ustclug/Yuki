package server

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Debug                 bool          `mapstructure:"debug"`
	DbURL                 string        `mapstructure:"db_url" validate:"required"`
	FileSystem            string        `mapstructure:"fs" validate:"oneof=xfs zfs default"`
	DockerEndpoint        string        `mapstructure:"docker_endpoint" validate:"unix_addr|tcp_addr"`
	Owner                 string        `mapstructure:"owner"`
	LogFile               string        `mapstructure:"log_file" validate:"filepath"`
	RepoLogsDir           string        `mapstructure:"repo_logs_dir" validate:"dir"`
	RepoConfigDir         []string      `mapstructure:"repo_config_dir" validate:"required,dive,dir"`
	LogLevel              string        `mapstructure:"log_level" validate:"oneof=debug info warn error"`
	ListenAddr            string        `mapstructure:"listen_addr" validate:"hostname_port"`
	BindIP                string        `mapstructure:"bind_ip" validate:"omitempty,ip"`
	NamePrefix            string        `mapstructure:"name_prefix"`
	PostSync              []string      `mapstructure:"post_sync"`
	ImagesUpgradeInterval time.Duration `mapstructure:"images_upgrade_interval" validate:"min=0"`
	SyncTimeout           time.Duration `mapstructure:"sync_timeout" validate:"min=0"`
}

func defaultDockerSocketLocation() string {
	// Add DOCKER_HOST (common convention) support for non-rootful-Docker implementation.
	if dockerHost, exists := os.LookupEnv("DOCKER_HOST"); exists {
		return dockerHost
	}
	return "unix:///var/run/docker.sock"
}

var DefaultConfig = Config{
	FileSystem:            "default",
	DockerEndpoint:        defaultDockerSocketLocation(),
	LogFile:               "/dev/stderr",
	Owner:                 fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
	RepoLogsDir:           "/var/log/yuki/",
	ListenAddr:            "127.0.0.1:9999",
	NamePrefix:            "syncing-",
	LogLevel:              "info",
	ImagesUpgradeInterval: time.Hour,
}
