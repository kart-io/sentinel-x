package middleware

import (
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareRecovery, func() MiddlewareConfig {
		return NewRecoveryOptions()
	})
}

// 确保 RecoveryOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*RecoveryOptions)(nil)

// RecoveryOptions defines recovery middleware options.
type RecoveryOptions struct {
	EnableStackTrace bool                                                       `json:"enable-stack-trace" mapstructure:"enable-stack-trace"`
	OnPanic          func(ctx transport.Context, err interface{}, stack []byte) `json:"-" mapstructure:"-"`
}

// NewRecoveryOptions creates default recovery middleware options.
func NewRecoveryOptions() *RecoveryOptions {
	return &RecoveryOptions{
		EnableStackTrace: false,
		OnPanic:          nil,
	}
}

// AddFlags adds flags for recovery options to the specified FlagSet.
func (o *RecoveryOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.BoolVar(&o.EnableStackTrace, options.Join(prefixes...)+"middleware.recovery.enable-stack-trace", o.EnableStackTrace, "Enable stack trace in error responses.")
}

// Complete completes the recovery options with defaults.
func (o *RecoveryOptions) Complete() error {
	return nil
}

// Validate validates the recovery options.
func (o *RecoveryOptions) Validate() []error {
	if o == nil {
		return nil
	}
	return nil
}
