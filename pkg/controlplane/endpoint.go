package controlplane

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

const unixBaseURL = "http://unix"

type EndpointType string

const (
	EndpointTCP  EndpointType = "tcp"
	EndpointUnix EndpointType = "unix"
)

type Endpoint struct {
	Type    EndpointType
	Address string
	BaseURL string
}

func ParseEndpoint(raw string) (Endpoint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Endpoint{}, fmt.Errorf("empty control plane endpoint")
	}

	if strings.HasPrefix(raw, "http://") {
		u, err := url.Parse(raw)
		if err != nil {
			return Endpoint{}, fmt.Errorf("parse control plane endpoint: %w", err)
		}
		if u.Host == "" || u.Path != "" && u.Path != "/" {
			return Endpoint{}, fmt.Errorf("invalid control plane endpoint %q", raw)
		}
		raw = u.Host
	}

	if filepath.IsAbs(raw) {
		return Endpoint{
			Type:    EndpointUnix,
			Address: raw,
			BaseURL: unixBaseURL,
		}, nil
	}

	if strings.Contains(raw, "/") {
		return Endpoint{}, fmt.Errorf("invalid control plane endpoint %q", raw)
	}

	if !strings.Contains(raw, ":") {
		return Endpoint{}, fmt.Errorf("invalid control plane endpoint %q", raw)
	}

	return Endpoint{
		Type:    EndpointTCP,
		Address: raw,
		BaseURL: "http://" + raw,
	}, nil
}
