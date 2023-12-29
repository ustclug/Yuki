package api

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

type ListReposResponseItem struct {
	Name       string `json:"name"`
	Interval   string `json:"interval"`
	Image      string `json:"image"`
	StorageDir string `json:"storageDir"`
}

type ListReposResponse = []ListReposResponseItem

type GetRepoLogsRequest struct {
	N    int `query:"n" validate:"min=0"`
	Tail int `query:"tail" validate:"min=0"`
}
