package middleware

import (
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/spf13/pflag"
)

// RecoveryOptions defines recovery middleware options.
// RecoveryOptions defines recovery middleware options.
type RecoveryOptions struct {
	EnableStackTrace bool                                                       `json:"enable-stack-trace" mapstructure:"enable-stack-trace"`
	OnPanic          func(ctx transport.Context, err interface{}, stack []byte) `json:"-" mapstructure:"-"`
}

func NewRecoveryOptions() *RecoveryOptions {
	return &RecoveryOptions{
		EnableStackTrace: false,
		OnPanic:          nil,
	}
}

func (o *RecoveryOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.EnableStackTrace, "middleware.recovery.enable-stack-trace", o.EnableStackTrace, "Enable stack trace in error responses")
}

func (o *RecoveryOptions) Complete() error {
	return nil
}

func (o *RecoveryOptions) Validate() error {
	return nil
}
