package factory

import (
	"encoding/json"
	"io"

	"github.com/go-resty/resty/v2"
)

type Factory interface {
	RESTClient() *resty.Client
	JSONEncoder(w io.Writer) *json.Encoder
}
