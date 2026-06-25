package factory

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/pflag"

	"github.com/ustclug/Yuki/pkg/controlplane"
)

type factoryImpl struct {
	remote string
}

func (f *factoryImpl) RESTClient() *resty.Client {
	endpoint, err := controlplane.ParseEndpoint(f.remote)
	if err != nil {
		panic(err)
	}

	cli := resty.New().SetBaseURL(endpoint.BaseURL)
	return cli.SetTransport(newHTTPTransport(endpoint))
}

func (f *factoryImpl) JSONEncoder(w io.Writer) *json.Encoder {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder
}

func New(flags *pflag.FlagSet) Factory {
	s := factoryImpl{}
	flags.StringVarP(&s.remote, "remote", "r", "/run/yuki/yukid.sock", "Remote address")
	return &s
}

func newHTTPTransport(endpoint controlplane.Endpoint) *http.Transport {
	if endpoint.Type != controlplane.EndpointUnix {
		return &http.Transport{}
	}

	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", endpoint.Address)
		},
	}
}
