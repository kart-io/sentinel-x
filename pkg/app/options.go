package app

import "github.com/spf13/pflag"

// CliOptions is the interface for CLI options.
// Any options struct implementing this interface can be used with App.
type CliOptions interface {
	// AddFlags adds flags to the flagset.
	AddFlags(fs *pflag.FlagSet)
	// Validate validates the options.
	Validate() error
	// Complete completes the options with defaults.
	Complete() error
}

// CompletableOptions is an optional interface for options that need completion.
type CompletableOptions interface {
	Complete() error
}

// ValidatableOptions is an optional interface for options that need validation.
type ValidatableOptions interface {
	Validate() error
}

// PrintableOptions is an optional interface for options that can print themselves.
type PrintableOptions interface {
	String() string
}
