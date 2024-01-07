package yukictl

import (
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/yukictl/cmd"
	"github.com/ustclug/Yuki/pkg/yukictl/cmd/meta"
	"github.com/ustclug/Yuki/pkg/yukictl/cmd/repo"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

func Register(root *cobra.Command, f factory.Factory) {
	root.AddCommand(
		cmd.NewCmdCompletion(),
		cmd.NewCmdReload(f),
		cmd.NewCmdSync(f),
		meta.NewCmdMeta(f),
		repo.NewCmdRepo(f),
	)
}
