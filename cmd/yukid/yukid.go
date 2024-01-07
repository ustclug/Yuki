package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/info"
	"github.com/ustclug/Yuki/pkg/server"
)

func main() {
	var (
		configPath   string
		printVersion bool
	)
	cmd := cobra.Command{
		Use:          "yukid",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if printVersion {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(info.VersionInfo)
			}

			s, err := server.New(configPath)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithCancel(context.Background())
			signals := make(chan os.Signal, 2)
			signal.Notify(signals, os.Interrupt)
			go func() {
				<-signals
				cancel()
				<-signals
				os.Exit(1)
			}()
			return s.Start(ctx)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "/etc/yuki/daemon.toml", "The path to config file")
	cmd.Flags().BoolVarP(&printVersion, "version", "V", false, "Print version information and quit")

	_ = cmd.Execute()
}
