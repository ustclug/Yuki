package ct

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/utils"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type rmOptions struct {
	name string
}

func (o *rmOptions) Complete(args []string) error {
	o.name = args[0]
	return nil
}

func (o *rmOptions) Run(f factory.Factory) error {
	u, err := f.MakeURL("api/v1/containers/%s", o.name)
	if err != nil {
		return err
	}
	var errMsg echo.HTTPError
	resp, err := f.RESTClient().R().
		SetError(&errMsg).
		Delete(u.String())
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", errMsg.Message)
	}
	fmt.Printf("Successfully delete: <%s>\n", o.name)
	return nil
}

func NewCmdContainerRemove(f factory.Factory) *cobra.Command {
	o := rmOptions{}
	cmd := &cobra.Command{
		Use:   "rm",
		Short: "Remove the given container",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(o.Complete(args))
			utils.CheckError(o.Run(f))
		},
	}
	return cmd
}
