package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/ustclug/Yuki/pkg/server"
)

func main() {
	s, err := server.New()
	if err != nil {
		log.Fatal(err)
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
	s.Start(ctx)
}
