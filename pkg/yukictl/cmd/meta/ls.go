package meta

import (
	"fmt"
	"os"
	"time"

	"github.com/docker/go-units"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/tabwriter"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type lsOptions struct {
	name string
}

func (o *lsOptions) Run(f factory.Factory) error {
	var (
		err    error
		errMsg echo.HTTPError
	)
	req := f.RESTClient().R().SetError(&errMsg)
	encoder := f.JSONEncoder(os.Stdout)
	if len(o.name) > 0 {
		var result api.GetRepoMetaResponse
		resp, err := req.
			SetResult(&result).
			SetPathParam("name", o.name).
			Get("api/v1/metas/{name}")
		if err != nil {
			return err
		}
		if resp.IsError() {
			return fmt.Errorf("%s", errMsg.Message)
		}
		return encoder.Encode(result)
	}

	var result api.ListRepoMetasResponse
	resp, err := req.SetResult(&result).Get("api/v1/metas")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", errMsg.Message)
	}
	tw := tabwriter.New(os.Stdout)
	tw.SetHeader([]string{"name", "upstream", "syncing", "size", "last-success", "next-run"})
	for _, r := range result {
		lastSuccess := ""
		nextRun := ""
		if r.LastSuccess > 0 {
			lastSuccess = time.Unix(r.LastSuccess, 0).Format(time.RFC3339)
		}
		if r.NextRun > 0 {
			nextRun = time.Unix(r.NextRun, 0).Format(time.RFC3339)
		}
		tw.Append(
			r.Name,
			r.Upstream,
			r.Syncing,
			units.BytesSize(float64(r.Size)),
			lastSuccess,
			nextRun,
		)
	}
	return tw.Render()
}

func NewCmdMetaLs(f factory.Factory) *cobra.Command {
	o := lsOptions{}
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List one or all metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.name = args[0]
			}
			return o.Run(f)
		},
	}
	return cmd
}
