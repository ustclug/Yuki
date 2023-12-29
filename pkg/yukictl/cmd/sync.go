package cmd

import (
	"fmt"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type syncOptions struct {
	debug bool
	name  string
}

func (o *syncOptions) Run(f factory.Factory) error {
	req := f.RESTClient().R()
	var errMsg echo.HTTPError
	resp, err := req.
		SetError(&errMsg).
		SetQueryParam("debug", strconv.FormatBool(o.debug)).
		SetPathParam("name", o.name).
		Post("api/v1/repos/{name}/sync")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", errMsg.Message)
	}

	fmt.Printf("Syncing <%s>\n", o.name)
	return nil
}

func NewCmdSync(f factory.Factory) *cobra.Command {
	o := syncOptions{}
	cmd := &cobra.Command{
		Use:     "sync",
		Args:    cobra.ExactArgs(1),
		Example: "  yukictl sync REPO",
		Short:   "Sync local repository with remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.name = args[0]
			return o.Run(f)
		},
	}
	cmd.Flags().BoolVarP(&o.debug, "debug", "v", false, "Debug mode")
	return cmd
}
