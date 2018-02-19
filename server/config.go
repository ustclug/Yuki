package server

import (
	"fmt"
	"os"
	"time"

	"github.com/knight42/Yuki/auth"
	"github.com/knight42/Yuki/core"
	"github.com/knight42/Yuki/fs"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type AppConfig struct {
	Debug bool `mapstructure:"debug,omitempty" validate:"-"`
	// DbURL contains username and password
	DbURL          string `mapstructure:"db_url,omitempty" validate:"omitempty,mongodb"`
	DbName         string `mapstructure:"db_name,omitempty" validate:"omitempty,alpha"`
	FileSystem     string `mapstructure:"fs,omitempty" validate:"omitempty,eq=xfs|eq=zfs|eq=default"`
	SessionAge     string `mapstructure:"session_age,omitempty" validate:"omitempty,duration"`
	DockerEndpoint string `mapstructure:"docker_endpoint,omitempty" validate:"omitempty,unix_addr|tcp_addr"`

	Owner                 string   `mapstructure:"owner,omitempty" validate:"-"`
	LogDir                string   `mapstructure:"log_dir,omitempty" validate:"-"`
	LogLevel              string   `mapstructure:"log_level,omitempty" validate:"omitempty,eq=debug|eq=info|eq=warn|eq=error"`
	ListenAddr            string   `mapstructure:"listen_addr,omitempty" validate:"omitempty,hostport"`
	BindIP                string   `mapstructure:"bind_ip,omitempty" validate:"omitempty,ip"`
	NamePrefix            string   `mapstructure:"name_prefix,omitempty" validate:"-"`
	SyncTimeout           string   `mapstructure:"sync_timeout,omitempty" validate:"omitempty,duration"`
	AuthProvider          string   `mapstructure:"auth_provider,omitempty" validate:"omitempty,eq=ldap|eq=none"`
	CookieKey             string   `mapstructure:"cookie_key,omitempty" validate:"required"`
	CookieDomain          string   `mapstructure:"cookie_domain,omitempty" validate:"-"`
	SecureCookie          bool     `mapstructure:"secure_cookie,omitempty" validate:"-"`
	PostSync              []string `mapstructure:"post_sync,omitempty" validate:"-"`
	AllowOrigins          []string `mapstructure:"allow_origins,omitempty" validate:"-"`
	ImagesUpgradeInterval string   `mapstructure:"images_upgrade_interval,omitempty" validate:"omitempty,cron"`

	Ldap auth.LdapAuthConfig `mapstructure:"ldap,omitempty" validate:"-"`
}

type Config struct {
	core.Config
	Owner                 string
	LogDir                string
	LogLevel              logrus.Level
	ListenAddr            string
	BindIP                string
	NamePrefix            string
	Authenticator         auth.Authenticator
	SyncTimeout           time.Duration
	CookieKey             string
	CookieDomain          string
	SecureCookie          bool
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
			SessionAge:     time.Hour * 24 * 2, // 2 days
			DockerEndpoint: "unix:///var/run/docker.sock",
		},
		Owner:                 fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
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
			DockerEndpoint: appCfg.DockerEndpoint,
		},
		Owner:                 appCfg.Owner,
		LogDir:                appCfg.LogDir,
		ListenAddr:            appCfg.ListenAddr,
		BindIP:                appCfg.BindIP,
		NamePrefix:            appCfg.NamePrefix,
		AllowOrigins:          appCfg.AllowOrigins,
		PostSync:              appCfg.PostSync,
		CookieKey:             appCfg.CookieKey,
		CookieDomain:          appCfg.CookieDomain,
		SecureCookie:          appCfg.SecureCookie,
		ImagesUpgradeInterval: appCfg.ImagesUpgradeInterval,
	}

	switch appCfg.AuthProvider {
	case "ldap":
		a, err := auth.NewLdapAuthenticator(&appCfg.Ldap)
		if err != nil {
			return nil, err
		}
		cfg.Authenticator = a
	default:
		cfg.Authenticator = auth.NewNopAuthenticator()
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

	dur, _ := time.ParseDuration(appCfg.SyncTimeout)
	cfg.SyncTimeout = dur

	dur, _ = time.ParseDuration(appCfg.SessionAge)
	cfg.SessionAge = dur
	return &cfg, nil
}
