package core

import (
	"os"
	"log"
	"testing"
	"io/ioutil"
)

var (
	C *Core
	LogDir string
)

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
	LogDir, err = ioutil.TempDir("", "yuki")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(LogDir)
	C.repoColl.RemoveAll(nil)
	C.metaColl.RemoveAll(nil)
	os.Exit(m.Run())
}
