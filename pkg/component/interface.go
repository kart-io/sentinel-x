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
//	func (o *MySQLOptions) Validate() error {
//	    if o.Host == "" {
//	        return fmt.Errorf("host is required")
//	    }
//	    return nil
//	}
//
//	func (o *MySQLOptions) AddFlags(fs *pflag.FlagSet) {
//	    fs.StringVar(&o.Host, "mysql.host", o.Host, "MySQL host")
//	    fs.IntVar(&o.Port, "mysql.port", o.Port, "MySQL port")
//	}
type ConfigOptions interface {
	// Complete fills in any fields not set that are required to have valid data.
	// This method should set default values for optional fields and derive
	// computed fields from other configuration.
	//
	// Returns an error if completion fails (e.g., unable to derive required values).
	Complete() error

	// Validate validates the options and returns an error if any option is invalid.
	// This method should check:
	//   - Required fields are populated
	//   - Field values are within acceptable ranges
	//   - Field combinations are logically consistent
	//
	// Validate should be called after Complete() to ensure all fields are properly set.
	// Returns nil if all validations pass.
	Validate() error

	// AddFlags adds flags for the options to the specified FlagSet.
	// This allows the configuration to be populated from command-line arguments.
	//
	// The fs parameter is the flag set to which flags should be added.
	// The namePrefix parameter is prepended to flag names to avoid conflicts
	// (e.g., "mysql." results in flags like "--mysql.host", "--mysql.port").
	// Implementations should use meaningful flag names and provide clear descriptions.
	AddFlags(fs *pflag.FlagSet, namePrefix string)
}
