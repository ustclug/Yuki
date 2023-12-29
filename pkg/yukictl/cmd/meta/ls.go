package meta

import (
	"fmt"
	"os"
	"time"

	"github.com/docker/go-units"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type outputMeta struct {
	api.Meta    `json:",inline"`
	Size        string     `json:"size"`
	LastSuccess *time.Time `json:"lastSuccess,omitempty"`
	CreatedAt   *struct{}  `json:"createdAt,omitempty"` // ignore
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
	PrevRun     *time.Time `json:"prevRun,omitempty"`
	NextRun     *time.Time `json:"nextRun,omitempty"`
}

func (o *outputMeta) From(m api.Meta) *outputMeta {
	o.Meta = m
	o.Size = units.BytesSize(float64(m.Size))
	if m.LastSuccess > 0 {
		t := time.Unix(m.LastSuccess, 0)
		o.LastSuccess = &t
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
	return o
}

type lsOptions struct {
	name string
}

func (o *lsOptions) Run(f factory.Factory) error {
	var (
		err    error
		errMsg echo.HTTPError
	)
	req := f.RESTClient().R().SetError(&errMsg)
	encoder := f.JSONEncoder(os.Stdout)
	if len(o.name) > 0 {
		var result api.Meta
		resp, err := req.
			SetResult(&result).
			SetPathParam("name", o.name).
			Get("api/v1/metas/{name}")
		if err != nil {
			return err
		}
		if resp.IsError() {
			return fmt.Errorf("%s", errMsg.Message)
		}
		var out outputMeta
		return encoder.Encode(out.From(result))
	}
	var result []api.Meta
	resp, err := req.SetResult(&result).Get("api/v1/metas")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", errMsg.Message)
	}
	outs := make([]outputMeta, 0, len(result))
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.name = args[0]
			}
			return o.Run(f)
		},
	}
	return cmd
}
