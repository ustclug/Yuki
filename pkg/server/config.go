package server

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"

	"github.com/ustclug/Yuki/pkg/fs"
)

type AppConfig struct {
	Debug             bool          `mapstructure:"debug"`
	DbURL             string        `mapstructure:"db_url" validate:"required"`
	FileSystem        string        `mapstructure:"fs" validate:"oneof=xfs zfs default"`
	DockerEndpoint    string        `mapstructure:"docker_endpoint" validate:"unix_addr|tcp_addr"`
	Owner             string        `mapstructure:"owner"`
	LogFile           string        `mapstructure:"log_file" validate:"filepath"`
	RepoLogsDir       string        `mapstructure:"repo_logs_dir" validate:"dirpath"`
	RepoConfigDir     []string      `mapstructure:"repo_config_dir" validate:"required"`
	LogLevel          string        `mapstructure:"log_level" validate:"oneof=debug info warn error"`
	ListenAddr        string        `mapstructure:"listen_addr" validate:"hostname_port"`
	BindIP            string        `mapstructure:"bind_ip" validate:"omitempty,ip"`
	NamePrefix        string        `mapstructure:"name_prefix"`
	PostSync          []string      `mapstructure:"post_sync"`
	ImagesUpgradeCron string        `mapstructure:"images_upgrade_cron" validate:"cron"`
	SyncTimeout       time.Duration `mapstructure:"sync_timeout" validate:"min=0"`
	SeccompProfile    string        `mapstructure:"seccomp_profile" validate:"omitempty,filepath"`
}

type Config struct {
	Debug             bool
	DbURL             string
	DockerEndpoint    string
	Owner             string
	LogFile           string
	RepoLogsDir       string
	RepoConfigDir     []string
	LogLevel          slog.Level
	ListenAddr        string
	BindIP            string
	NamePrefix        string
	PostSync          []string
	ImagesUpgradeCron string
	SyncTimeout       time.Duration
	SeccompProfile    string
	GetSizer          fs.GetSizer
}

func loadConfig(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	appCfg := AppConfig{
		Debug:             false,
		FileSystem:        "default",
		DockerEndpoint:    "unix:///var/run/docker.sock",
		LogFile:           "/dev/stderr",
		Owner:             fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		RepoLogsDir:       "/var/log/yuki/",
		ListenAddr:        "127.0.0.1:9999",
		NamePrefix:        "syncing-",
		LogLevel:          "info",
		ImagesUpgradeCron: "@every 1h",
	}
	if err := v.Unmarshal(&appCfg); err != nil {
		return nil, err
	}
	validate := validator.New()
	if err := validate.Struct(&appCfg); err != nil {
		return nil, err
	}
	cfg := Config{
		Debug:             appCfg.Debug,
		DbURL:             appCfg.DbURL,
		DockerEndpoint:    appCfg.DockerEndpoint,
		Owner:             appCfg.Owner,
		LogFile:           appCfg.LogFile,
		RepoConfigDir:     appCfg.RepoConfigDir,
		RepoLogsDir:       appCfg.RepoLogsDir,
		ListenAddr:        appCfg.ListenAddr,
		BindIP:            appCfg.BindIP,
		NamePrefix:        appCfg.NamePrefix,
		PostSync:          appCfg.PostSync,
		ImagesUpgradeCron: appCfg.ImagesUpgradeCron,
		SyncTimeout:       appCfg.SyncTimeout,
		SeccompProfile:    appCfg.SeccompProfile,
	}

	switch appCfg.FileSystem {
	case "zfs":
		cfg.GetSizer = fs.New(fs.ZFS)
	case "xfs":
		cfg.GetSizer = fs.New(fs.XFS)
	default:
		cfg.GetSizer = fs.New(fs.DEFAULT)
	}

	switch appCfg.LogLevel {
	case "debug":
		cfg.LogLevel = slog.LevelDebug
	case "warn":
		cfg.LogLevel = slog.LevelWarn
	case "error":
		cfg.LogLevel = slog.LevelError
	default:
		cfg.LogLevel = slog.LevelInfo
	}

	return &cfg, nil
}
