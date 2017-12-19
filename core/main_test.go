package core

import (
	"os"
	"log"
	"testing"
)

var C *Core

func TestMain(m *testing.M) {
	var err error
	C, err = NewWithConfig(Config{
		DbURL: "127.0.0.1:27017",
		DbName: "test",
		DockerEndpoint: "unix:///var/run/docker.sock",
	})
	if err != nil {
		log.Fatal(err)
	}
	C.repoColl.RemoveAll(nil)
	C.metaColl.RemoveAll(nil)
	os.Exit(m.Run())
}
