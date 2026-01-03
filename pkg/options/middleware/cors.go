package middleware

import (
	"errors"

	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareCORS, func() MiddlewareConfig {
		return NewCORSOptions()
	})
}

// 确保 CORSOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*CORSOptions)(nil)

// CORSOptions defines CORS middleware options.
type CORSOptions struct {
	AllowOrigins     []string `json:"allow-origins" mapstructure:"allow-origins"`
	AllowMethods     []string `json:"allow-methods" mapstructure:"allow-methods"`
	AllowHeaders     []string `json:"allow-headers" mapstructure:"allow-headers"`
	ExposeHeaders    []string `json:"expose-headers" mapstructure:"expose-headers"`
	AllowCredentials bool     `json:"allow-credentials" mapstructure:"allow-credentials"`
	MaxAge           int      `json:"max-age" mapstructure:"max-age"`
}

// NewCORSOptions creates default CORS options.
func NewCORSOptions() *CORSOptions {
	return &CORSOptions{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{},
		AllowCredentials: false,
		MaxAge:           86400,
	}
}

// AddFlags adds flags for CORS options to the specified FlagSet.
func (o *CORSOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringSliceVar(&o.AllowOrigins, options.Join(prefixes...)+"middleware.cors.allow-origins", o.AllowOrigins, "CORS allowed origins.")
	fs.StringSliceVar(&o.AllowMethods, options.Join(prefixes...)+"middleware.cors.allow-methods", o.AllowMethods, "CORS allowed methods.")
	fs.StringSliceVar(&o.AllowHeaders, options.Join(prefixes...)+"middleware.cors.allow-headers", o.AllowHeaders, "CORS allowed headers.")
	fs.StringSliceVar(&o.ExposeHeaders, options.Join(prefixes...)+"middleware.cors.expose-headers", o.ExposeHeaders, "CORS exposed headers.")
	fs.BoolVar(&o.AllowCredentials, options.Join(prefixes...)+"middleware.cors.allow-credentials", o.AllowCredentials, "CORS allow credentials.")
	fs.IntVar(&o.MaxAge, options.Join(prefixes...)+"middleware.cors.max-age", o.MaxAge, "CORS preflight max age.")
}

// Validate validates the CORS options.
func (o *CORSOptions) Validate() []error {
	if o == nil {
		return nil
	}
	var errs []error
	if len(o.AllowOrigins) == 0 {
		errs = append(errs, errors.New("CORS: AllowOrigins must be explicitly configured, empty list not allowed"))
	}
	return errs
}

// Complete completes the CORS options with defaults.
func (o *CORSOptions) Complete() error {
	return nil
}
