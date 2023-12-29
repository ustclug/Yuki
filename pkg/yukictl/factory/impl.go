package factory

import (
	"encoding/json"
	"io"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/pflag"
)

type factoryImpl struct {
	remote string
}

func (f *factoryImpl) RESTClient() *resty.Client {
	return resty.New().SetBaseURL(f.remote)
}

func (f *factoryImpl) JSONEncoder(w io.Writer) *json.Encoder {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder
}

func New(flags *pflag.FlagSet) Factory {
	s := factoryImpl{}
	flags.StringVarP(&s.remote, "remote", "r", "http://127.0.0.1:9999/", "Remote address")
	return &s
}
