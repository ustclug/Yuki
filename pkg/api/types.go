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
	Cron       string `json:"cron"`
	Image      string `json:"image"`
	StorageDir string `json:"storageDir"`
}

type ListReposResponse = []ListReposResponseItem
