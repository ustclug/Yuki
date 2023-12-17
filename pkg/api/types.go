package api

import (
	"time"
)

// Container provides the ID and labels of a container.
type Container struct {
	ID     string
	Labels map[string]string
}

// ContainerDetail describes a container in detail.
type ContainerDetail struct {
	ID      string `json:"id"`
	Image   string `json:"image"`
	Created int64  `json:"created"`
	State   string `json:"state"`
	Status  string `json:"status"`
	Name    string `json:"name"`
}

// LogFileStat describes a log file.
type LogFileStat struct {
	Name  string    `json:"name"`
	Size  int64     `json:"size"`
	Mtime time.Time `json:"mtime"`
}

// Meta represents the metadata of a Repository.
type Meta struct {
	Name        string `bson:"_id" json:"name"`
	Upstream    string `bson:"-" json:"upstream"`
	Syncing     bool   `bson:"-" json:"syncing"`
	Size        int64  `bson:"size" json:"size"`
	ExitCode    int    `bson:"exitCode" json:"exitCode"`
	LastSuccess int64  `bson:"lastSuccess,omitempty" json:"lastSuccess"`
	CreatedAt   int64  `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt   int64  `bson:"updatedAt,omitempty" json:"updatedAt"`
	PrevRun     int64  `bson:"prevRun,omitempty" json:"prevRun"`
	NextRun     int64  `bson:"-" json:"nextRun"`
}

// M is an alias for a map[string]string map.
type M = map[string]string

// Repository contains a list of syncing options.
type Repository struct {
	Name        string `bson:"_id" json:"name" validate:"required"`
	Interval    string `bson:"interval" json:"interval" validate:"required,cron"`
	Image       string `bson:"image" json:"image" validate:"required,containsrune=:"`
	StorageDir  string `bson:"storageDir" json:"storageDir" validate:"required"`
	User        string `bson:"user,omitempty" json:"user,omitempty" validate:"omitempty,gt=1"`
	BindIP      string `bson:"bindIP,omitempty" json:"bindIP,omitempty" validate:"omitempty,ip"`
	Network     string `bson:"network,omitempty" json:"network,omitempty" validate:"omitempty"`
	LogRotCycle *int   `bson:"logRotCycle,omitempty" json:"logRotCycle,omitempty" validate:"omitempty,min=0"`
	Retry       int    `bson:"retry,omitempty" json:"retry,omitempty" validate:"min=0"`
	Envs        M      `bson:"envs,omitempty" json:"envs,omitempty" validate:"omitempty,dive,keys,required,endkeys,required"`
	Volumes     M      `bson:"volumes,omitempty" json:"volumes,omitempty" validate:"omitempty,dive,keys,required,endkeys,required"`
	CreatedAt   int64  `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	UpdatedAt   int64  `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

type RepoSummary struct {
	Name       string `bson:"_id" json:"name"`
	Interval   string `bson:"interval" json:"interval"`
	Image      string `bson:"image" json:"image"`
	StorageDir string `bson:"storageDir" json:"storageDir" validate:"required"`
}

type PreSyncPayload struct {
	Name string
}

type PostSyncPayload struct {
	ID       string
	Name     string
	Dir      string
	ExitCode int
}

type ListRepoMetasResponse = []GetMetaResponse

type GetMetaResponse struct {
	Name        string `json:"name"`
	Upstream    string `json:"upstream"`
	Syncing     bool   `json:"syncing"`
	Size        int64  `json:"size"`
	ExitCode    int    `json:"exitCode"`
	LastSuccess int64  `json:"lastSuccess"`
	UpdatedAt   int64  `json:"updatedAt"`
	PrevRun     int64  `json:"prevRun"`
	NextRun     int64  `json:"nextRun"`
}
