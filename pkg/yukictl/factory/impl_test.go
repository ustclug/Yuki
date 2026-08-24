package factory

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHTTPTransportTCP(t *testing.T) {
	cli, err := (&factoryImpl{remote: "127.0.0.1:9999"}).RESTClient()
	require.NoError(t, err)
	transport, ok := cli.GetClient().Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.Proxy)
}

func TestNewHTTPTransportUnix(t *testing.T) {
	cli, err := (&factoryImpl{remote: "/tmp/yukid.sock"}).RESTClient()
	require.NoError(t, err)
	transport, ok := cli.GetClient().Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.DialContext)
	require.NotNil(t, transport.Proxy)

	_, err = transport.DialContext(context.Background(), "tcp", "unused")
	var netErr *net.OpError
	require.ErrorAs(t, err, &netErr)
	require.Equal(t, "unix", netErr.Net)
}

func TestRESTClientInvalidEndpoint(t *testing.T) {
	cli, err := (&factoryImpl{remote: "invalid"}).RESTClient()
	require.Nil(t, cli)
	require.ErrorContains(t, err, "invalid control plane endpoint")
}
