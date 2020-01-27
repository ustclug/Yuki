package factory

import (
	"net/url"

	"github.com/go-resty/resty/v2"
)

type Factory interface {
	RESTClient() *resty.Client
	MakeURL(format string, a ...interface{}) (*url.URL, error)
}
