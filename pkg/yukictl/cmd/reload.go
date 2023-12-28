package cmd

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type reloadOptions struct {
	repo string
}

func (o *reloadOptions) Complete(args []string) error {
	if len(args) > 0 {
		o.repo = args[0]
	}
	return nil
}

func (o *reloadOptions) Run(f factory.Factory) error {
	req := f.RESTClient().R()
	u, err := f.MakeURL("api/v1/repositories")
	if err != nil {
		return err
	}
	if len(o.repo) > 0 {
		u.Path += "/" + o.repo
	}
	var errMsg echo.HTTPError
	resp, err := req.SetError(&errMsg).Post(u.String())
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", errMsg.Message)
	}
	if len(o.repo) > 0 {
		fmt.Printf("Successfully reloaded: <%s>\n", o.repo)
	} else {
		fmt.Println("Successfully reloaded all repositories")
	}
	return nil
}

func NewCmdReload(f factory.Factory) *cobra.Command {
	o := reloadOptions{}
	cmd := &cobra.Command{
		Use:   "reload [name]",
		Short: "Reload config of one or all repos",
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
