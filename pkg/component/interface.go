// Package component defines the component interfaces.
package component

import "github.com/spf13/pflag"

// ConfigOptions defines the standard interface for all component options.
// All component configuration types (MySQL, Redis, MongoDB, PostgreSQL, Etcd, etc.)
// must implement this interface to ensure consistent behavior across the system.
//
// This interface provides a unified contract for:
//   - Completing configuration with default values
//   - Validating configuration parameters
//   - Adding command-line flags
//
// Example implementation:
//
//	type MySQLOptions struct {
//	    Host     string
//	    Port     int
//	    Username string
//	    Password string
//	}
//
//	func (o *MySQLOptions) Complete() error {
//	    // Fill in default values if not set
//	    if o.Port == 0 {
//	        o.Port = 3306
//	    }
//	    return nil
//	}
//
//	func (o *MySQLOptions) Validate() []error {
//	    var errs []error
//	    if o.Host == "" {
//	        errs = append(errs, fmt.Errorf("host is required"))
//	    }
//	    return errs
//	}
//
//	func (o *MySQLOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
//	    prefix := ""
//	    if len(prefixes) > 0 {
//	        prefix = prefixes[0]
//	    }
//	    fs.StringVar(&o.Host, prefix+"host", o.Host, "MySQL host")
//	    fs.IntVar(&o.Port, prefix+"port", o.Port, "MySQL port")
//	}
type ConfigOptions interface {
	// Complete fills in any fields not set that are required to have valid data.
	// This method should set default values for optional fields and derive
	// computed fields from other configuration.
	//
	// Returns an error if completion fails (e.g., unable to derive required values).
	Complete() error

	// Validate validates the options and returns all validation errors.
	// This method should check:
	//   - Required fields are populated
	//   - Field values are within acceptable ranges
	//   - Field combinations are logically consistent
	//
	// Validate should be called after Complete() to ensure all fields are properly set.
	// Returns nil or empty slice if all validations pass.
	Validate() []error

	// AddFlags adds flags for the options to the specified FlagSet.
	// This allows the configuration to be populated from command-line arguments.
	//
	// The fs parameter is the flag set to which flags should be added.
	// The prefixes parameter allows prepending a prefix to flag names to avoid conflicts
	// (e.g., "mysql." results in flags like "--mysql.host", "--mysql.port").
	// Implementations should use meaningful flag names and provide clear descriptions.
	AddFlags(fs *pflag.FlagSet, prefixes ...string)
}
