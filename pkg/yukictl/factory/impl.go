package factory

import (
	"context"
	"encoding/json"
	"fmt"
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

func (f *factoryImpl) RESTClient() (*resty.Client, error) {
	endpoint, err := controlplane.ParseEndpoint(f.remote)
	if err != nil {
		return nil, err
	}

	cli := resty.New().SetBaseURL(endpoint.BaseURL)
	if endpoint.Type != controlplane.EndpointUnix {
		return cli, nil
	}
	transport, ok := cli.GetClient().Transport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("unsupported HTTP transport type %T", cli.GetClient().Transport)
	}
	transport = transport.Clone()
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", endpoint.Address)
	}
	return cli.SetTransport(transport), nil
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
