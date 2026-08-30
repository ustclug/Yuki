package server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/model"
)

func TestHandlerListRepoMetas(t *testing.T) {
	te := NewTestEnv(t)
	require.NoError(t, te.server.db.Create([]model.RepoMeta{
		{
			Name: "repo2",
		},
		{
			Name: "repo1",
		},
	}).Error)
	require.NoError(t, te.server.db.Create([]model.Repo{
		{
			Name: "repo1",
		},
		{
			Name:    "repo2",
			Mirrorz: []model.MirrorzRepo{},
		},
	}).Error)

	var metas api.ListRepoMetasResponse
	cli := te.RESTClient()
	resp, err := cli.R().SetResult(&metas).Get("/metas")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	require.Len(t, metas, 2)
	require.Equal(t, "repo1", metas[0].Name)
	require.Equal(t, []api.MirrorzRepo{{Name: "repo1"}}, metas[0].Mirrorz)
	require.Empty(t, metas[1].Mirrorz)
}

func TestHandlerGetRepoMeta(t *testing.T) {
	te := NewTestEnv(t)
	require.NoError(t, te.server.db.Create([]model.RepoMeta{
		{
			Name: t.Name(),
		},
	}).Error)
	require.NoError(t, te.server.db.Create(&model.Repo{
		Name: t.Name(),
		Mirrorz: []model.MirrorzRepo{
			{
				Name: "logical-repo",
				Desc: "A logical repository",
			},
		},
	}).Error)

	cli := te.RESTClient()
	testCases := map[string]struct {
		name         string
		expectStatus int
	}{
		"ok": {
			name:         t.Name(),
			expectStatus: http.StatusOK,
		},
		"not found": {
			name:         "not found",
			expectStatus: http.StatusNotFound,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var meta api.GetRepoMetaResponse
			resp, err := cli.R().SetResult(&meta).Get("/metas/" + tc.name)
			require.NoError(t, err)
			require.Equal(t, tc.expectStatus, resp.StatusCode())
			if tc.expectStatus == http.StatusOK {
				require.Equal(t, []api.MirrorzRepo{{
					Name: "logical-repo",
					Desc: "A logical repository",
				}}, meta.Mirrorz)
			}
		})
	}
}
