package main

import (
	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/yukictl"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
	"github.com/ustclug/Yuki/pkg/yukictl/globalflag"
)

func main() {
	rootCmd := &cobra.Command{
		Use: "yukictl",
	}
	gflag := globalflag.New()
	gflag.AddFlags(rootCmd.PersistentFlags())
	f := factory.New(gflag)
	yukictl.Register(rootCmd, f)
	_ = rootCmd.Execute()
}
