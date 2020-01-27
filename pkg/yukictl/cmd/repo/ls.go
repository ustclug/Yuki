package repo

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/utils"
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
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(result)
		return err
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
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(result)
	return err
}

func NewCmdRepoLs(f factory.Factory) *cobra.Command {
	o := repoLsOptions{}
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List one or all repositories",
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(o.Complete(args))
			utils.CheckError(o.Run(f))
		},
	}
	return cmd
}
