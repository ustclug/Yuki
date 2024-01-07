package repo

import (
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

func NewCmdRepo(f factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories",
	}
	cmd.AddCommand(
		NewCmdRepoLs(f),
		NewCmdRepoRm(f),
	)
	return cmd
}
