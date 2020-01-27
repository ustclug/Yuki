package meta

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

type lsOptions struct {
	name string
}

func (o *lsOptions) Complete(args []string) error {
	if len(args) > 0 {
		o.name = args[0]
	}
	return nil
}

func (o *lsOptions) Run(f factory.Factory) error {
	var (
		u      *url.URL
		err    error
		errMsg echo.HTTPError
	)
	req := f.RESTClient().R().SetError(&errMsg)
	if len(o.name) > 0 {
		u, err = f.MakeURL("api/v1/metas/%s", o.name)
		if err != nil {
			return err
		}
		var result api.Meta
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
	u, err = f.MakeURL("api/v1/metas")
	if err != nil {
		return err
	}
	var result []api.Meta
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

func NewCmdMetaLs(f factory.Factory) *cobra.Command {
	o := lsOptions{}
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List one or all metadata",
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(o.Complete(args))
			utils.CheckError(o.Run(f))
		},
	}
	return cmd
}
