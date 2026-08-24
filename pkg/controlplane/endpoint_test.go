package controlplane

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    Endpoint
		wantErr string
	}{
		{
			name: "tcp endpoint",
			raw:  "127.0.0.1:9999",
			want: Endpoint{
				Type:    EndpointTCP,
				Address: "127.0.0.1:9999",
				BaseURL: "http://127.0.0.1:9999",
			},
		},
		{
			name: "legacy http endpoint",
			raw:  "http://127.0.0.1:9999/",
			want: Endpoint{
				Type:    EndpointTCP,
				Address: "127.0.0.1:9999",
				BaseURL: "http://127.0.0.1:9999",
			},
		},
		{
			name: "IPv6 endpoint",
			raw:  "[::1]:9999",
			want: Endpoint{
				Type:    EndpointTCP,
				Address: "[::1]:9999",
				BaseURL: "http://[::1]:9999",
			},
		},
		{
			name: "unix endpoint",
			raw:  "/run/yuki/yukid.sock",
			want: Endpoint{
				Type:    EndpointUnix,
				Address: "/run/yuki/yukid.sock",
				BaseURL: unixBaseURL,
			},
		},
		{
			name:    "empty endpoint",
			raw:     " ",
			wantErr: "empty control plane endpoint",
		},
		{
			name:    "relative path not allowed",
			raw:     "run/yuki.sock",
			wantErr: "invalid control plane endpoint",
		},
		{
			name:    "missing port",
			raw:     "localhost",
			wantErr: "invalid control plane endpoint",
		},
		{
			name:    "non-numeric port",
			raw:     "localhost:abc",
			wantErr: "invalid control plane endpoint",
		},
		{
			name:    "IPv6 missing port",
			raw:     "[::1]",
			wantErr: "invalid control plane endpoint",
		},
		{
			name:    "empty port",
			raw:     "127.0.0.1:",
			wantErr: "invalid control plane endpoint",
		},
		{
			name:    "port out of range",
			raw:     "127.0.0.1:65536",
			wantErr: "invalid control plane endpoint",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseEndpoint(tc.raw)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
