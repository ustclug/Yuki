package globalflag

import (
	"fmt"
	"net/url"
	"path"

	"github.com/spf13/pflag"
)

type FlagSet struct {
	remote string
}

func (f *FlagSet) MakeURL(format string, args ...interface{}) (*url.URL, error) {
	p := fmt.Sprintf(format, args...)
	u, err := url.Parse(f.remote)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %s", err)
	}
	u.Path = path.Join(u.Path, p)
	return u, nil
}

func (f *FlagSet) AddFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&f.remote, "remote", "r", "http://127.0.0.1:9999/", "Remote address")
}

func New() *FlagSet {
	return &FlagSet{}
}
