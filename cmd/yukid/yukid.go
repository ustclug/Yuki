package main

import (
	"log"

	"github.com/ustclug/Yuki/pkg/server"
)

func main() {
	s, err := server.New()
	if err != nil {
		log.Fatal(err)
	}
	s.Start()
}
