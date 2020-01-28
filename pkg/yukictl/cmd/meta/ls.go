package meta

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/utils"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type outputMeta struct {
	api.Meta    `json:",inline"`
	LastSuccess *time.Time `json:"lastSuccess,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
	PrevRun     *time.Time `json:"prevRun,omitempty"`
	NextRun     *time.Time `json:"nextRun,omitempty"`
}

func (o *outputMeta) From(m api.Meta) {
	o.Meta = m
	if m.LastSuccess > 0 {
		t := time.Unix(m.LastSuccess, 0)
		o.LastSuccess = &t
	}
	if m.CreatedAt > 0 {
		t := time.Unix(m.CreatedAt, 0)
		o.CreatedAt = &t
	}
	if m.UpdatedAt > 0 {
		t := time.Unix(m.UpdatedAt, 0)
		o.UpdatedAt = &t
	}
	if m.PrevRun > 0 {
		t := time.Unix(m.PrevRun, 0)
		o.PrevRun = &t
	}
	if m.NextRun > 0 {
		t := time.Unix(m.NextRun, 0)
		o.NextRun = &t
	}
}

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
	encoder := f.JSONEncoder(os.Stdout)
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
		var out outputMeta
		out.From(result)
		return encoder.Encode(out)
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
	var outs []outputMeta
	for _, r := range result {
		var out outputMeta
		out.From(r)
		outs = append(outs, out)
	}
	return encoder.Encode(outs)
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
