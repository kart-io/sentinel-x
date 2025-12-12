package middleware

// CORSOptions defines CORS middleware options.
type CORSOptions struct {
	AllowOrigins     []string `json:"allow-origins" mapstructure:"allow-origins"`
	AllowMethods     []string `json:"allow-methods" mapstructure:"allow-methods"`
	AllowHeaders     []string `json:"allow-headers" mapstructure:"allow-headers"`
	ExposeHeaders    []string `json:"expose-headers" mapstructure:"expose-headers"`
	AllowCredentials bool     `json:"allow-credentials" mapstructure:"allow-credentials"`
	MaxAge           int      `json:"max-age" mapstructure:"max-age"`
}

// WithCORS configures and enables CORS middleware.
func WithCORS(origins []string, methods []string, headers []string, credentials bool, maxAge int) Option {
	return func(o *Options) {
		o.DisableCORS = false
		if origins != nil {
			o.CORS.AllowOrigins = origins
		}
		if methods != nil {
			o.CORS.AllowMethods = methods
		}
		if headers != nil {
			o.CORS.AllowHeaders = headers
		}
		o.CORS.AllowCredentials = credentials
		if maxAge > 0 {
			o.CORS.MaxAge = maxAge
		}
	}
}

// WithoutCORS disables CORS middleware.
func WithoutCORS() Option {
	return func(o *Options) { o.DisableCORS = true }
}
