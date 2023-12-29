package repo

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type logsOptions struct {
	name string
	tail uint8
	nth  uint8
}

func (o *logsOptions) Run(cmd *cobra.Command, f factory.Factory) error {
	flags := cmd.Flags()

	var errMsg echo.HTTPError
	req := f.RESTClient().R().SetError(&errMsg)

	if flags.Changed("nth") {
		req.SetQueryParam("n", strconv.FormatUint(uint64(o.nth), 10))
	}
	if flags.Changed("tail") {
		req.SetQueryParam("tail", strconv.FormatUint(uint64(o.tail), 10))
	}

	resp, err := req.SetDoNotParseResponse(true).
		SetPathParam("name", o.name).
		Get("api/v1/repos/{name}/logs")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", errMsg.Message)
	}
	body := resp.RawBody()
	defer body.Close()
	_, err = io.Copy(os.Stdout, body)
	return err
}

func NewCmdRepoLogs(f factory.Factory) *cobra.Command {
	o := logsOptions{}
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View logs of the given repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.name = args[0]
			return o.Run(cmd, f)
		},
	}
	flags := cmd.Flags()
	flags.Uint8Var(&o.tail, "tail", 0, "Output the last N lines")
	flags.Uint8VarP(&o.nth, "nth", "n", 0, "View the nth log file")
	return cmd
}
