package core

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
)

var (
	c      *Core
	logDir string
)

func TestMain(m *testing.M) {
	var err error
	c, err = NewWithConfig(Config{
		DbURL:          "127.0.0.1:27017",
		DbName:         "test",
		DockerEndpoint: "unix:///var/run/docker.sock",
	})
	if err != nil {
		log.Fatal(err)
	}
	logDir, err = ioutil.TempDir("", "yuki")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(logDir)
	_ = c.repoColl.DropCollection()
	_ = c.metaColl.DropCollection()
	os.Exit(m.Run())
}
