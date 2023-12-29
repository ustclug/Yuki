package repo

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type rmOptions struct {
	name string
}

func (o *rmOptions) Run(f factory.Factory) error {
	var errMsg echo.HTTPError
	resp, err := f.RESTClient().R().
		SetError(&errMsg).
		SetPathParam("name", o.name).
		Delete("api/v1/repos/{name}")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", errMsg.Message)
	}
	fmt.Printf("Successfully deleted from database: <%s>\n", o.name)
	return nil
}

func NewCmdRepoRm(f factory.Factory) *cobra.Command {
	o := rmOptions{}
	return &cobra.Command{
		Use:     "rm",
		Short:   "Remove repository from database",
		Example: "  yukictl repo rm REPO",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.name = args[0]
			return o.Run(f)
		},
	}
}
