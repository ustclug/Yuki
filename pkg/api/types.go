package api

type ListRepoMetasResponse = []GetRepoMetaResponse

type GetRepoMetaResponse struct {
	Name        string        `json:"name"`
	Upstream    string        `json:"upstream"`
	Syncing     bool          `json:"syncing"`
	Size        int64         `json:"size"`
	ExitCode    int           `json:"exitCode"`
	LastSuccess int64         `json:"lastSuccess"`
	UpdatedAt   int64         `json:"updatedAt"`
	PrevRun     int64         `json:"prevRun"`
	NextRun     int64         `json:"nextRun"`
	Mirrorz     []MirrorzRepo `json:"mirrorz"`
}

type MirrorzRepo struct {
	Name     string `json:"name"`
	CName    string `json:"cname,omitempty"`
	Desc     string `json:"desc,omitempty"`
	URL      string `json:"url,omitempty"`
	Help     string `json:"help,omitempty"`
	Upstream string `json:"upstream,omitempty"`
	Disable  bool   `json:"disable,omitempty"`
}

type ListReposResponseItem struct {
	Name       string `json:"name"`
	Cron       string `json:"cron"`
	Image      string `json:"image"`
	StorageDir string `json:"storageDir"`
}

type ListReposResponse = []ListReposResponseItem
