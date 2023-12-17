package model

type StringMap map[string]string

// Repo represents a Repository.
type Repo struct {
	Name        string `gorm:"primaryKey"`
	Interval    string
	Image       string
	StorageDir  string
	User        string
	BindIP      string
	Network     string
	LogRotCycle int
	Retry       int
	Envs        StringMap `gorm:"type:text;serializer:json"`
	Volumes     StringMap `gorm:"type:text;serializer:json"`
	// sqlite3 does not have builtin datetime type
	CreatedAt int64 `gorm:"autoCreateTime"`
	UpdatedAt int64 `gorm:"autoUpdateTime"`
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
