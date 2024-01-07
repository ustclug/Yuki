package server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ustclug/Yuki/pkg/model"
)

func TestHandlerListRepoMetas(t *testing.T) {
	te := NewTestEnv(t)
	require.NoError(t, te.server.db.Create([]model.RepoMeta{
		{
			Name: t.Name(),
		},
	}).Error)

	var metas []model.RepoMeta
	cli := te.RESTClient()
	resp, err := cli.R().SetResult(&metas).Get("/metas")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess(), "Unexpected response: %s", resp.Body())

	require.Len(t, metas, 1)
	require.EqualValues(t, t.Name(), metas[0].Name)
}

func TestHandlerGetRepoMeta(t *testing.T) {
	te := NewTestEnv(t)
	require.NoError(t, te.server.db.Create([]model.RepoMeta{
		{
			Name: t.Name(),
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
			resp, err := cli.R().Get("/metas/" + tc.name)
			require.NoError(t, err)
			require.EqualValues(t, tc.expectStatus, resp.StatusCode())
		})
	}
}
