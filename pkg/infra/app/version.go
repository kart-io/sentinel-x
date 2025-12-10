// Package app provides application bootstrapping with Cobra, Viper, and Pflag.
package app

import (
	"github.com/kart-io/version"
	"github.com/spf13/pflag"
)

// GetVersion returns the version string.
func GetVersion() string {
	return version.Get().GitVersion
}

// GetVersionInfo returns the full version information.
func GetVersionInfo() version.Info {
	return version.Get()
}

// AddVersionFlags adds version-related flags to the flagset.
func AddVersionFlags(fs *pflag.FlagSet) {
	version.AddFlags(fs)
}

// PrintAndExitIfRequested prints version and exits if --version flag is set.
func PrintAndExitIfRequested() {
	version.PrintAndExitIfRequested()
}
