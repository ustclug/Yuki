package server

import (
	"fmt"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ustclug/Yuki/pkg/core"
	"github.com/ustclug/Yuki/pkg/fs"
)

type AppConfig struct {
	Debug bool `mapstructure:"debug,omitempty" validate:"-"`
	// DbURL contains username and password
	DbURL          string `mapstructure:"db_url,omitempty" validate:"required"`
	DbName         string `mapstructure:"db_name,omitempty" validate:"omitempty,alpha"`
	FileSystem     string `mapstructure:"fs,omitempty" validate:"omitempty,oneof=xfs zfs default"`
	DockerEndpoint string `mapstructure:"docker_endpoint,omitempty" validate:"omitempty,unix_addr|tcp_addr"`

	Owner                 string        `mapstructure:"owner,omitempty" validate:"-"`
	LogDir                string        `mapstructure:"log_dir,omitempty" validate:"-"`
	RepoConfigDir         []string      `mapstructure:"repo_config_dir,omitempty" validate:"required"`
	LogLevel              string        `mapstructure:"log_level,omitempty" validate:"omitempty,oneof=debug info warn error"`
	ListenAddr            string        `mapstructure:"listen_addr,omitempty" validate:"omitempty,hostname_port"`
	BindIP                string        `mapstructure:"bind_ip,omitempty" validate:"omitempty,ip"`
	NamePrefix            string        `mapstructure:"name_prefix,omitempty" validate:"-"`
	PostSync              []string      `mapstructure:"post_sync,omitempty" validate:"-"`
	ImagesUpgradeInterval string        `mapstructure:"images_upgrade_interval,omitempty" validate:"omitempty,cron"`
	SyncTimeout           time.Duration `mapstructure:"sync_timeout,omitempty" validate:"omitempty,gte=0"`
	SeccompProfile        string        `mapstructure:"seccomp_profile,omitempty" validate:"-"`
}

type Config struct {
	core.Config
	Owner                 string
	LogDir                string
	RepoConfigDir         []string
	LogLevel              logrus.Level
	ListenAddr            string
	BindIP                string
	NamePrefix            string
	PostSync              []string
	ImagesUpgradeInterval string
	SyncTimeout           time.Duration
	SeccompProfile        string
}

var (
	DefaultServerConfig = Config{
		Config: core.Config{
			Debug:          false,
			DbURL:          "127.0.0.1:27017",
			DbName:         "mirror",
			DockerEndpoint: "unix:///var/run/docker.sock",
		},
		Owner:                 fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		LogDir:                "/var/log/yuki/",
		ListenAddr:            "127.0.0.1:9999",
		NamePrefix:            "syncing-",
		LogLevel:              logrus.InfoLevel,
		ImagesUpgradeInterval: "@every 1h",
		SyncTimeout:           0,
		SeccompProfile:        "",
	}
)

func LoadConfig() (*Config, error) {
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	appCfg := new(AppConfig)
	if err := viper.Unmarshal(appCfg); err != nil {
		return nil, err
	}
	validate := validator.New()
	if err := validate.Struct(appCfg); err != nil {
		return nil, err
	}
	cfg := Config{
		Config: core.Config{
			Debug:          appCfg.Debug,
			DbURL:          appCfg.DbURL,
			DbName:         appCfg.DbName,
			DockerEndpoint: appCfg.DockerEndpoint,
		},
		Owner:                 appCfg.Owner,
		RepoConfigDir:         appCfg.RepoConfigDir,
		LogDir:                appCfg.LogDir,
		ListenAddr:            appCfg.ListenAddr,
		BindIP:                appCfg.BindIP,
		NamePrefix:            appCfg.NamePrefix,
		PostSync:              appCfg.PostSync,
		ImagesUpgradeInterval: appCfg.ImagesUpgradeInterval,
		SyncTimeout:           appCfg.SyncTimeout,
		SeccompProfile:        appCfg.SeccompProfile,
	}

	switch appCfg.FileSystem {
	case "zfs":
		cfg.Config.GetSizer = fs.New(fs.ZFS)
	case "xfs":
		cfg.Config.GetSizer = fs.New(fs.XFS)
	default:
		cfg.Config.GetSizer = fs.New(fs.DEFAULT)
	}

	switch appCfg.LogLevel {
	case "debug":
		cfg.LogLevel = logrus.DebugLevel
	case "warn":
		cfg.LogLevel = logrus.WarnLevel
	case "error":
		cfg.LogLevel = logrus.ErrorLevel
	case "info":
		fallthrough
	default:
		cfg.LogLevel = logrus.InfoLevel
	}

	return &cfg, nil
}
