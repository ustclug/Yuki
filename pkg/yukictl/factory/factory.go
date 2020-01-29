package factory

import (
	"encoding/json"
	"io"
	"net/url"

	"github.com/go-resty/resty/v2"
)

type Factory interface {
	RESTClient() *resty.Client
	JSONEncoder(w io.Writer) *json.Encoder
	MakeURL(format string, a ...interface{}) (*url.URL, error)
}
