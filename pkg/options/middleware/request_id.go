package middleware

// RequestIDOptions defines request ID middleware options.
type RequestIDOptions struct {
	Header    string        `json:"header" mapstructure:"header"`
	Generator func() string `json:"-" mapstructure:"-"`
}

// WithRequestID configures request ID middleware.
func WithRequestID(header string, generator func() string) Option {
	return func(o *Options) {
		o.DisableRequestID = false
		if header != "" {
			o.RequestID.Header = header
		}
		if generator != nil {
			o.RequestID.Generator = generator
		}
	}
}

// WithoutRequestID disables request ID middleware.
func WithoutRequestID() Option {
	return func(o *Options) { o.DisableRequestID = true }
}
