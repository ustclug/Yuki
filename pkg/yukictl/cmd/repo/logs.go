package repo

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/docker/go-units"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/tabwriter"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type logsOptions struct {
	name  string
	tail  uint8
	nth   uint8
	stats bool
}

func (o *logsOptions) Complete(cmd *cobra.Command, args []string) error {
	o.name = args[0]
	flags := cmd.Flags()
	if flags.Changed("stats") && (flags.Changed("tail") || flags.Changed("nth")) {
		return fmt.Errorf("--stats cannot be used with --tail or --nth")
	}
	return nil
}

func (o *logsOptions) Run(cmd *cobra.Command, f factory.Factory) error {
	u, err := f.MakeURL("api/v1/repositories/%s/logs", o.name)
	if err != nil {
		return err
	}

	flags := cmd.Flags()

	var errMsg echo.HTTPError
	req := f.RESTClient().R().SetError(&errMsg)

	if o.stats {
		var stats []api.LogFileStat
		resp, err := req.
			SetQueryParam("stats", strconv.FormatBool(true)).
			SetResult(&stats).
			Get(u.String())
		if err != nil {
			return err
		}
		if resp.IsError() {
			return fmt.Errorf("%s", errMsg.Message)
		}
		printer := tabwriter.New(os.Stdout)
		printer.SetHeader([]string{"name", "mtime", "size"})
		for _, s := range stats {
			printer.Append(s.Name, s.Mtime.Format(time.RFC3339), units.BytesSize(float64(s.Size)))
		}
		return printer.Render()
	}

	if flags.Changed("nth") {
		req.SetQueryParam("n", strconv.FormatUint(uint64(o.nth), 10))
	}
	if flags.Changed("tail") {
		req.SetQueryParam("tail", strconv.FormatUint(uint64(o.tail), 10))
	}

	resp, err := req.SetDoNotParseResponse(true).Get(u.String())
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
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.Complete(cmd, args)
			if err != nil {
				return err
			}
			return o.Run(cmd, f)
		},
	}
	flags := cmd.Flags()
	flags.Uint8Var(&o.tail, "tail", 0, "Output the last N lines")
	flags.Uint8VarP(&o.nth, "nth", "n", 0, "View the nth log file")
	flags.BoolVar(&o.stats, "stats", false, "Get the information of log files")
	return cmd
}
