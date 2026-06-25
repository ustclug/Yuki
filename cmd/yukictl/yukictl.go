package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/info"
	"github.com/ustclug/Yuki/pkg/yukictl"
	"github.com/ustclug/Yuki/pkg/yukictl/factory"
)

func main() {
	var printVersion bool
	rootCmd := &cobra.Command{
		Use:          "yukictl",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if printVersion {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(info.VersionInfo)
			}
			return nil
		},
	}
	rootCmd.Flags().BoolVarP(&printVersion, "version", "V", false, "Print version information and quit")
	f := factory.New(rootCmd.PersistentFlags())
	yukictl.Register(rootCmd, f)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
