package model

type StringMap map[string]string

// Repo represents a Repository.
type Repo struct {
	Name string `gorm:"primaryKey" json:"name" validate:"required"`
	// TODO: rename to Cron?
	Interval    string    `json:"interval" validate:"required,cron"`
	Image       string    `json:"image" validate:"required,containsrune=:"`
	StorageDir  string    `json:"storageDir" validate:"required,dir"`
	User        string    `json:"user" validate:"omitempty,gt=1"`
	BindIP      string    `json:"bindIP" validate:"omitempty,ip"`
	Network     string    `json:"network"`
	LogRotCycle int       `json:"logRotCycle" validate:"min=0"`
	Retry       int       `json:"retry"  validate:"min=0"`
	Envs        StringMap `gorm:"type:text;serializer:json" json:"envs"`
	Volumes     StringMap `gorm:"type:text;serializer:json" json:"volumes"`
	// sqlite3 does not have builtin datetime type
	CreatedAt int64 `gorm:"autoCreateTime" json:"-"`
	UpdatedAt int64 `gorm:"autoUpdateTime" json:"-"`
}

// RepoMeta represents the metadata of a Repository.
type RepoMeta struct {
	Name        string `gorm:"primaryKey"`
	Upstream    string
	Size        int64
	ExitCode    int
	CreatedAt   int64 `gorm:"autoCreateTime"`
	UpdatedAt   int64 `gorm:"autoUpdateTime"`
	LastSuccess int64
	PrevRun     int64
}
