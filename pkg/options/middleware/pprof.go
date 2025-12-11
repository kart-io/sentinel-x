package middleware

// PprofOptions defines pprof options.
type PprofOptions struct {
	Prefix               string `json:"prefix" mapstructure:"prefix"`
	EnableCmdline        bool   `json:"enable-cmdline" mapstructure:"enable-cmdline"`
	EnableProfile        bool   `json:"enable-profile" mapstructure:"enable-profile"`
	EnableSymbol         bool   `json:"enable-symbol" mapstructure:"enable-symbol"`
	EnableTrace          bool   `json:"enable-trace" mapstructure:"enable-trace"`
	BlockProfileRate     int    `json:"block-profile-rate" mapstructure:"block-profile-rate"`
	MutexProfileFraction int    `json:"mutex-profile-fraction" mapstructure:"mutex-profile-fraction"`
}

// WithPprof configures and enables pprof endpoints.
func WithPprof(prefix string, blockRate, mutexFraction int) Option {
	return func(o *Options) {
		o.DisablePprof = false
		if prefix != "" {
			o.Pprof.Prefix = prefix
		}
		if blockRate >= 0 {
			o.Pprof.BlockProfileRate = blockRate
		}
		if mutexFraction >= 0 {
			o.Pprof.MutexProfileFraction = mutexFraction
		}
	}
}

// WithoutPprof disables pprof endpoints.
func WithoutPprof() Option {
	return func(o *Options) { o.DisablePprof = true }
}
