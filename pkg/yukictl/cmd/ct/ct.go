package ct

import (
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

func NewCmdContainer(f factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ct",
		Short: "Manage containers",
	}
	cmd.AddCommand(
		NewCmdContainerLs(f),
		NewCmdContainerLogs(f),
		NewCmdContainerRemove(f),
	)
	return cmd
}
