package factory

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ustclug/Yuki/pkg/controlplane"
)

func TestNewHTTPTransportTCP(t *testing.T) {
	transport := newHTTPTransport(controlplane.Endpoint{Type: controlplane.EndpointTCP})
	require.Nil(t, transport.DialContext)
}

func TestNewHTTPTransportUnix(t *testing.T) {
	transport := newHTTPTransport(controlplane.Endpoint{
		Type:    controlplane.EndpointUnix,
		Address: "/tmp/yukid.sock",
	})
	require.NotNil(t, transport.DialContext)

	_, err := transport.DialContext(context.Background(), "tcp", "unused")
	var netErr *net.OpError
	require.ErrorAs(t, err, &netErr)
	require.Equal(t, "unix", netErr.Net)
}
