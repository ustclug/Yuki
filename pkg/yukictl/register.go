package yukictl

import (
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/yukictl/cmd"
	"github.com/ustclug/Yuki/pkg/yukictl/cmd/ct"
	"github.com/ustclug/Yuki/pkg/yukictl/cmd/meta"
	"github.com/ustclug/Yuki/pkg/yukictl/cmd/repo"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

func Register(root *cobra.Command, f factory.Factory) {
	root.AddCommand(
		ct.NewCmdContainer(f),
		cmd.NewCmdExport(f),
		meta.NewCmdMeta(f),
		cmd.NewCmdReload(f),
		repo.NewCmdRepo(f),
		cmd.NewCmdSync(f),
	)
}
