package main

import (
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/yukictl"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

func main() {
	rootCmd := &cobra.Command{
		Use:          "yukictl",
		SilenceUsage: true,
	}
	f := factory.New(rootCmd.PersistentFlags())
	yukictl.Register(rootCmd, f)
	_ = rootCmd.Execute()
}
