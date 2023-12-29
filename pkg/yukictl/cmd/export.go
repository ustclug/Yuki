package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

type exportOptions struct {
	names []string
	dir   string
}

func (o *exportOptions) Run(f factory.Factory) error {
	req := f.RESTClient().R()
	if len(o.names) > 0 {
		req.SetQueryParam("names", strings.Join(o.names, ","))
	}
	var (
		repos  []api.Repository
		errMsg echo.HTTPError
	)
	resp, err := req.
		SetResult(&repos).
		SetError(&errMsg).
		Get("api/v1/config")
	if err != nil {
		return fmt.Errorf("send request: %s", err)
	}
	if resp.IsError() {
		return fmt.Errorf("%s", errMsg.Message)
	}
	if len(o.dir) > 0 {
		for _, r := range repos {
			data, _ := yaml.Marshal(r)
			err := os.WriteFile(filepath.Join(o.dir, r.Name+".yaml"), data, 0644)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return f.JSONEncoder(os.Stdout).Encode(repos)
}

func NewCmdExport(f factory.Factory) *cobra.Command {
	o := &exportOptions{}
	cmd := &cobra.Command{
		Use:   "export [name]",
		Short: "Export config",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.names = args
			return o.Run(f)
		},
	}
	cmd.Flags().StringVarP(&o.dir, "dir", "d", "", "Dest directory")
	return cmd
}
