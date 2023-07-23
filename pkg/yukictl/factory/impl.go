package factory

import (
	"encoding/json"
	"io"

	"github.com/go-resty/resty/v2"

	"github.com/ustclug/Yuki/pkg/yukictl/globalflag"
)

type factoryImpl struct {
	*globalflag.FlagSet
}

func (f *factoryImpl) RESTClient() *resty.Client {
	return resty.New()
}

func (f *factoryImpl) JSONEncoder(w io.Writer) *json.Encoder {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder
}

func New(flags *globalflag.FlagSet) Factory {
	s := factoryImpl{
		FlagSet: flags,
	}
	return &s
}
