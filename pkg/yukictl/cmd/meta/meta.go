package meta

import (
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

func NewCmdMeta(f factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "meta",
		Short: "List metas",
	}
	cmd.AddCommand(
		NewCmdMetaLs(f),
	)
	return cmd
}
