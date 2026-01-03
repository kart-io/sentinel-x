package middleware

import (
	"errors"

	"github.com/google/uuid"
	"github.com/spf13/pflag"
)

// RequestIDOptions defines request ID middleware options.
type RequestIDOptions struct {
	Header    string        `json:"header" mapstructure:"header"`
	Generator func() string `json:"-" mapstructure:"-"`
}

func NewRequestIDOptions() *RequestIDOptions {
	return &RequestIDOptions{
		Header: "X-Request-ID",
		Generator: func() string {
			return uuid.New().String()
		},
	}
}

func (o *RequestIDOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Header, "middleware.request-id.header", o.Header, "Request ID header name")
}

func (o *RequestIDOptions) Validate() error {
	if o.Header == "" {
		return errors.New("request ID header name is required")
	}
	return nil
}

func (o *RequestIDOptions) Complete() error {
	return nil
}
