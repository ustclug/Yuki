package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/ustclug/Yuki/pkg/server"
)

func main() {
	var configPath string
	cmd := cobra.Command{
		Use:          "yukid",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
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

	_ = cmd.Execute()
}
