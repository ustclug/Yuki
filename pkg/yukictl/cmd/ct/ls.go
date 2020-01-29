package ct

import (
	"fmt"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/tabwriter"
	"github.com/ustclug/Yuki/pkg/utils"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type lsOptions struct {
}

func (o *lsOptions) Run(f factory.Factory) error {
	var errMsg echo.HTTPError
	req := f.RESTClient().R().SetError(&errMsg)
	u, err := f.MakeURL("api/v1/containers")
	if err != nil {
		return err
	}
	var result []api.ContainerDetail
	resp, err := req.SetResult(&result).Get(u.String())
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", errMsg.Message)
	}
	printer := tabwriter.New(os.Stdout)
	printer.SetHeader([]string{
		"id",
		"name",
		"image",
		"state",
		"status",
	})
	for _, ct := range result {
		printer.Append(ct.ID, ct.Name, ct.Image, ct.State, ct.Status)
	}
	return printer.Render()
}

func NewCmdContainerLs(f factory.Factory) *cobra.Command {
	o := lsOptions{}
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List all containers",
		Args:  cobra.MaximumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(o.Run(f))
		},
	}
	return cmd
}
