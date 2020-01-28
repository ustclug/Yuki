package ct

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

type logsOptions struct {
	name   string
	tail   uint32
	follow bool
}

func (o *logsOptions) Complete(args []string) error {
	o.name = args[0]
	return nil
}

func (o *logsOptions) Run(f factory.Factory) error {
	u, err := f.MakeURL("api/v1/containers/%s/logs", o.name)
	if err != nil {
		return err
	}
	var errMsg echo.HTTPError
	req := f.RESTClient().R().SetError(&errMsg).SetDoNotParseResponse(true)
	if o.follow {
		req.SetQueryParam("follow", strconv.FormatBool(true))
	}
	if o.tail > 0 {
		req.SetQueryParam("tail", strconv.FormatUint(uint64(o.tail), 10))
	}
	resp, err := req.Get(u.String())
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", errMsg.Message)
	}
	body := resp.RawBody()
	defer body.Close()
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, body)
	return err
}

func NewCmdContainerLogs(f factory.Factory) *cobra.Command {
	o := logsOptions{}
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Container logs",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(o.Complete(args))
			utils.CheckError(o.Run(f))
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&o.follow, "follow", "f", false, "Follow log output")
	flags.Uint32VarP(&o.tail, "tail", "t", 0, "Number of lines to show from the end of the logs")
	return cmd
}
