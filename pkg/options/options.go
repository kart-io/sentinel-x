// Package options defines the generic options interface and common utilities.
package options

import (
	"strings"

	"github.com/spf13/pflag"
)

// Join concatenates prefixes with "." separator.
// If the result is non-empty, it appends a trailing ".".
// This is used to build flag names like "mysql.host" or "prefix.mysql.host".
func Join(prefixes ...string) string {
	joined := strings.Join(prefixes, ".")
	if joined != "" {
		joined += "."
	}
	return joined
}

// IOptions defines methods to implement a generic options.
type IOptions interface {
	// Validate validates all the required options.
	// It can also used to complete options if needed.
	Validate() []error

	// AddFlags adds flags related to given flagset.
	AddFlags(fs *pflag.FlagSet, prefixes ...string)
}
