package api

type ListRepoMetasResponse = []GetRepoMetaResponse

type GetRepoMetaResponse struct {
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
