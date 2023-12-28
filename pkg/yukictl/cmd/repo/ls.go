package repo

import (
	"fmt"
	"net/url"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/tabwriter"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type repoLsOptions struct {
	name string
}

func (o *repoLsOptions) Complete(args []string) error {
	if len(args) > 0 {
		o.name = args[0]
	}
	return nil
}

func (o *repoLsOptions) Run(f factory.Factory) error {
	var (
		u      *url.URL
		err    error
		errMsg echo.HTTPError
	)
	req := f.RESTClient().R().SetError(&errMsg)
	encoder := f.JSONEncoder(os.Stdout)
	if len(o.name) > 0 {
		u, err = f.MakeURL("api/v1/repositories/%s", o.name)
		if err != nil {
			return err
		}
		var result api.Repository
		resp, err := req.SetResult(&result).Get(u.String())
		if err != nil {
			return err
		}
		if resp.IsError() {
			return fmt.Errorf("%s", errMsg.Message)
		}
		return encoder.Encode(result)
	}
	u, err = f.MakeURL("api/v1/repositories")
	if err != nil {
		return err
	}
	var result []api.RepoSummary
	resp, err := req.SetResult(&result).Get(u.String())
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
			err := o.Complete(args)
			if err != nil {
				return err
			}
			return o.Run(f)
		},
	}
	return cmd
}
