package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/utils"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type syncOptions struct {
	debug bool
	name  string
}

func (o *syncOptions) Complete(args []string) error {
	o.name = args[0]
	return nil
}

func (o *syncOptions) Run(f factory.Factory) error {
	req := f.RESTClient().R()
	u, err := f.MakeURL("api/v1/containers/%s", o.name)
	if err != nil {
		return err
	}
	var errMsg echo.HTTPError
	resp, err := req.
		SetError(&errMsg).
		SetQueryParam("debug", strconv.FormatBool(o.debug)).
		Post(u.String())
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", errMsg.Message)
	}

	if !o.debug {
		fmt.Printf("Syncing <%s>\n", o.name)
		return nil
	}

	u.Path += "/logs"
	resp, err = f.RESTClient().R().
		SetError(&errMsg).
		SetDoNotParseResponse(true).
		SetQueryParam("follow", strconv.FormatBool(true)).
		Get(u.String())
	if err != nil {
		return err
	}
	body := resp.RawBody()
	defer body.Close()
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, body)
	return err
}

func NewCmdSync(f factory.Factory) *cobra.Command {
	o := syncOptions{}
	cmd := &cobra.Command{
		Use:     "sync",
		Args:    cobra.MinimumNArgs(1),
		Example: "  yukictl sync REPO",
		Short:   "Sync local repository with remote",
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(o.Complete(args))
			utils.CheckError(o.Run(f))
		},
	}
	cmd.Flags().BoolVar(&o.debug, "debug", false, "Debug mode")
	return cmd
}
