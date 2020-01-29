package factory

import (
	"encoding/json"
	"io"

	"github.com/go-resty/resty/v2"

	"github.com/ustclug/Yuki/pkg/yukictl/globalflag"
)

type fatoryImpl struct {
	*globalflag.FlagSet
}

func (f *fatoryImpl) RESTClient() *resty.Client {
	return resty.New()
}

func (f *fatoryImpl) JSONEncoder(w io.Writer) *json.Encoder {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder
}

func New(flags *globalflag.FlagSet) Factory {
	s := fatoryImpl{
		FlagSet: flags,
	}
	return &s
}
