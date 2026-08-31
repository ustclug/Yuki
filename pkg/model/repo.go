package model

import "fmt"

type StringMap map[string]string

// MirrorzRepo describes a logical repository to which a sync task contributes.
// Runtime state remains attached to RepoMeta; consumers are responsible for
// aggregating multiple tasks that reference the same logical repository.
type MirrorzRepo struct {
	Name     string `json:"name" validate:"required"`
	CName    string `json:"cname,omitempty"`
	Desc     string `json:"desc,omitempty"`
	URL      string `json:"url,omitempty"`
	Help     string `json:"help,omitempty"`
	Upstream string `json:"upstream,omitempty"`
	Disable  bool   `json:"disable,omitempty"`
}

// Repo represents a Repository.
type Repo struct {
	Name string `gorm:"primaryKey" json:"name" validate:"required,repo-name"`
	// NOTE: the cron validator does not support */number syntax
	Cron        string        `json:"cron" validate:"required"`
	Image       string        `json:"image" validate:"required"`
	StorageDir  string        `json:"storageDir" validate:"required,dir"`
	User        string        `json:"user"`
	BindIP      string        `json:"bindIP" validate:"omitempty,ip"`
	Network     string        `json:"network"`
	LogRotCycle int           `json:"logRotCycle" validate:"min=0"`
	Retry       int           `json:"retry"  validate:"min=0"`
	Envs        StringMap     `gorm:"type:text;serializer:json" json:"envs"`
	Volumes     StringMap     `gorm:"type:text;serializer:json" json:"volumes"`
	Mirrorz     []MirrorzRepo `gorm:"type:text;serializer:json" json:"mirrorz" validate:"dive"`
	// sqlite3 does not have builtin datetime type
	CreatedAt int64 `gorm:"autoCreateTime" json:"-"`
	UpdatedAt int64 `gorm:"autoUpdateTime" json:"-"`
}

// NormalizeMirrorz applies the default one-task-to-one-repository mapping.
// A non-nil empty slice explicitly opts the task out of MirrorZ integration.
func (r *Repo) NormalizeMirrorz() error {
	if r.Mirrorz == nil {
		r.Mirrorz = []MirrorzRepo{{Name: r.Name}}
	}

	seen := make(map[string]struct{}, len(r.Mirrorz))
	for i := range r.Mirrorz {
		if r.Mirrorz[i].Name == "" {
			r.Mirrorz[i].Name = r.Name
		}
		name := r.Mirrorz[i].Name
		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate MirrorZ repository %q", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

// EffectiveMirrorz returns the persisted mapping, applying the default for
// databases created before the MirrorZ field was introduced.
func (r Repo) EffectiveMirrorz() []MirrorzRepo {
	if r.Mirrorz == nil {
		return []MirrorzRepo{{Name: r.Name}}
	}
	return r.Mirrorz
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
	NextRun     int64
	Syncing     bool
}
