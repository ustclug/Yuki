package server

import (
	"time"

	"github.com/knight42/Yuki/core"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type AppConfig struct {
	Debug bool `mapstructure:"debug,omitempty" validate:"-"`
	// DbURL contains username and password
	DbURL          string `mapstructure:"db_url,omitempty" validate:"required,mongodb"`
	DbName         string `mapstructure:"db_name,omitempty" validate:"alpha"`
	FileSystem     string `mapstructure:"fs,omitempty" validate:"omitempty,eq=xfs|eq=zfs"`
	DockerEndpoint string `mapstructure:"docker_endpoint,omitempty" validate:"required,unix_addr|tcp_addr"`

	Owner                 string   `mapstructure:"owner,omitempty" validate:"-"`
	LogDir                string   `mapstructure:"log_dir,omitempty" validate:"required"`
	LogLevel              string   `mapstructure:"log_level,omitempty" validate:"omitempty,eq=debug|eq=info|eq=warn|eq=error"`
	ListenAddr            string   `mapstructure:"listen_addr,omitempty" validate:"hostport,required"`
	BindIP                string   `mapstructure:"bind_ip,omitempty" validate:"omitempty,ip"`
	NamePrefix            string   `mapstructure:"name_prefix,omitempty" validate:"-"`
	SyncTimeout           string   `mapstructure:"sync_timeout,omitempty" validate:"omitempty,duration"`
	PostSync              []string `mapstructure:"post_sync,omitempty" validate:"-"`
	AllowOrigins          []string `mapstructure:"allow_origins,omitempty" validate:"-"`
	ImagesUpgradeInterval string   `mapstructure:"images_upgrade_interval,omitempty" validate:"required,cron"`
}

type Config struct {
	core.Config
	Owner                 string
	LogDir                string
	LogLevel              logrus.Level
	ListenAddr            string
	BindIP                string
	NamePrefix            string
	SyncTimeout           time.Duration
	AllowOrigins          []string
	PostSync              []string
	ImagesUpgradeInterval string
}

var (
	DefaultServerConfig = Config{
		Config: core.Config{
			Debug:          false,
			DbURL:          "127.0.0.1:27017",
			DbName:         "mirror",
			FileSystem:     "",
			DockerEndpoint: "unix:///var/run/docker.sock",
		},
		Owner:                 "0:0",
		LogDir:                "/var/log/yuki/",
		ListenAddr:            "127.0.0.1:9999",
		NamePrefix:            "syncing-",
		LogLevel:              logrus.InfoLevel,
		AllowOrigins:          []string{"*"},
		ImagesUpgradeInterval: "@every 1h",
	}
)

func LoadConfig() (*Config, error) {
	configValidator := NewValidator()
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	appCfg := new(AppConfig)
	if err := viper.Unmarshal(appCfg); err != nil {
		return nil, err
	}
	if err := configValidator.Struct(appCfg); err != nil {
		return nil, err
	}
	cfg := Config{
		Config: core.Config{
			Debug:          appCfg.Debug,
			DbURL:          appCfg.DbURL,
			DbName:         appCfg.DbName,
			FileSystem:     appCfg.FileSystem,
			DockerEndpoint: appCfg.DockerEndpoint,
		},
		Owner:                 appCfg.Owner,
		LogDir:                appCfg.LogDir,
		ListenAddr:            appCfg.ListenAddr,
		BindIP:                appCfg.BindIP,
		NamePrefix:            appCfg.NamePrefix,
		AllowOrigins:          appCfg.AllowOrigins,
		PostSync:              appCfg.PostSync,
		ImagesUpgradeInterval: appCfg.ImagesUpgradeInterval,
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

	dur, _ := time.ParseDuration(appCfg.SyncTimeout)
	cfg.SyncTimeout = dur
	return &cfg, nil
}
