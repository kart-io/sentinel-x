package middleware

import (
	"errors"

	"github.com/spf13/pflag"
)

// CORSOptions defines CORS middleware options.
type CORSOptions struct {
	AllowOrigins     []string `json:"allow-origins" mapstructure:"allow-origins"`
	AllowMethods     []string `json:"allow-methods" mapstructure:"allow-methods"`
	AllowHeaders     []string `json:"allow-headers" mapstructure:"allow-headers"`
	ExposeHeaders    []string `json:"expose-headers" mapstructure:"expose-headers"`
	AllowCredentials bool     `json:"allow-credentials" mapstructure:"allow-credentials"`
	MaxAge           int      `json:"max-age" mapstructure:"max-age"`
}

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

func (o *CORSOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.AllowOrigins, "middleware.cors.allow-origins", o.AllowOrigins, "CORS allowed origins")
	fs.StringSliceVar(&o.AllowMethods, "middleware.cors.allow-methods", o.AllowMethods, "CORS allowed methods")
	fs.StringSliceVar(&o.AllowHeaders, "middleware.cors.allow-headers", o.AllowHeaders, "CORS allowed headers")
	fs.StringSliceVar(&o.ExposeHeaders, "middleware.cors.expose-headers", o.ExposeHeaders, "CORS exposed headers")
	fs.BoolVar(&o.AllowCredentials, "middleware.cors.allow-credentials", o.AllowCredentials, "CORS allow credentials")
	fs.IntVar(&o.MaxAge, "middleware.cors.max-age", o.MaxAge, "CORS preflight max age")
}

func (o *CORSOptions) Validate() error {
	if len(o.AllowOrigins) == 0 {
		return errors.New("CORS: AllowOrigins must be explicitly configured, empty list not allowed")
	}
	return nil
}

func (o *CORSOptions) Complete() error {
	return nil
}
