package repo

import (
	"fmt"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/model"
	"github.com/ustclug/Yuki/pkg/tabwriter"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type repoLsOptions struct {
	name string
}

func (o *repoLsOptions) Run(f factory.Factory) error {
	var (
		err    error
		errMsg echo.HTTPError
	)
	req := f.RESTClient().R().SetError(&errMsg)
	encoder := f.JSONEncoder(os.Stdout)
	if len(o.name) > 0 {
		var result model.Repo
		resp, err := req.SetResult(&result).SetPathParam("name", o.name).Get("api/v1/repos/{name}")
		if err != nil {
			return err
		}
		if resp.IsError() {
			return fmt.Errorf("%s", errMsg.Message)
		}
		return encoder.Encode(result)
	}

	var result api.ListReposResponse
	resp, err := req.SetResult(&result).Get("api/v1/repos")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", errMsg.Message)
	}
	printer := tabwriter.New(os.Stdout)
	printer.SetHeader([]string{
		"name",
		"interval",
		"image",
		"storage-dir",
	})
	for _, r := range result {
		printer.Append(r.Name, r.Interval, r.Image, r.StorageDir)
	}
	return printer.Render()
}

func NewCmdRepoLs(f factory.Factory) *cobra.Command {
	o := repoLsOptions{}
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List one or all repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.name = args[0]
			}
			return o.Run(f)
		},
	}
	return cmd
}
