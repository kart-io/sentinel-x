// Package app provides application bootstrapping with Cobra, Viper, and Pflag.
//
// This package provides a unified way to:
//   - Define CLI commands with Cobra
//   - Load configuration from files, environment variables, and flags using Viper
//   - Use the functional options pattern for configuration
//
// Usage:
//
//	app := app.NewApp(
//	    app.WithName("myapp"),
//	    app.WithDescription("My application"),
//	    app.WithOptions(opts),
//	    app.WithRunFunc(run),
//	)
//	app.Run()
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	options "github.com/kart-io/sentinel-x/pkg/options/app"
	"github.com/kart-io/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// App is the main application structure.
type App struct {
	name        string
	shortDesc   string
	description string
	options     options.CliOptions
	runFunc     RunFunc
	cmd         *cobra.Command
	args        cobra.PositionalArgs
	silence     bool
	noVersion   bool
	noConfig    bool
}

// RunFunc is the application's run function.
type RunFunc func() error

// Option configures an App.
type Option func(*App)

// WithName sets the application name.
func WithName(name string) Option {
	return func(a *App) {
		a.name = name
	}
}

// WithShortDescription sets the short description.
func WithShortDescription(desc string) Option {
	return func(a *App) {
		a.shortDesc = desc
	}
}

// WithDescription sets the long description.
func WithDescription(desc string) Option {
	return func(a *App) {
		a.description = desc
	}
}

// WithOptions sets the CLI options.
func WithOptions(opts options.CliOptions) Option {
	return func(a *App) {
		a.options = opts
	}
}

// WithRunFunc sets the run function.
func WithRunFunc(run RunFunc) Option {
	return func(a *App) {
		a.runFunc = run
	}
}

// WithArgs sets the positional args validation.
func WithArgs(args cobra.PositionalArgs) Option {
	return func(a *App) {
		a.args = args
	}
}

// WithSilence disables usage and error printing.
func WithSilence() Option {
	return func(a *App) {
		a.silence = true
	}
}

// WithNoVersion disables version flag.
func WithNoVersion() Option {
	return func(a *App) {
		a.noVersion = true
	}
}

// WithNoConfig disables config file loading.
func WithNoConfig() Option {
	return func(a *App) {
		a.noConfig = true
	}
}

// NewApp creates a new application instance.
func NewApp(opts ...Option) *App {
	a := &App{
		name: filepath.Base(os.Args[0]),
	}

	for _, opt := range opts {
		opt(a)
	}

	a.buildCommand()
	return a
}

// buildCommand creates the cobra command.
func (a *App) buildCommand() {
	cmd := &cobra.Command{
		Use:   a.name,
		Short: a.shortDesc,
		Long:  a.description,
		RunE:  a.runCommand,
		Args:  a.args,
		// Always silence usage on errors - users can use --help to see usage
		SilenceUsage: true,
	}

	if a.silence {
		cmd.SilenceErrors = true
	}

	// Add flags
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.Flags().SortFlags = true

	// Add global flags
	a.addGlobalFlags(cmd)

	// Add options flags using NamedFlagSets
	if a.options != nil {
		fss := a.options.Flags()
		for _, name := range fss.Order {
			fs := fss.FlagSets[name]
			cmd.Flags().AddFlagSet(fs)
		}
	}

	a.cmd = cmd
}

// addGlobalFlags adds global flags to the command.
func (a *App) addGlobalFlags(cmd *cobra.Command) {
	if !a.noConfig {
		cmd.PersistentFlags().StringP("config", "c", "", "Path to config file")
	}

	if !a.noVersion {
		// Use kart-io/version package for version flags
		version.AddFlags(cmd.PersistentFlags())
	}

	// Add help flag
	cmd.PersistentFlags().BoolP("help", "h", false, "Help for "+a.name)
}

// runCommand is the main run function for the command.
func (a *App) runCommand(cmd *cobra.Command, _ []string) error {
	// Handle version flag - this will print and exit if --version is set
	if !a.noVersion {
		version.PrintAndExitIfRequested()
	}

	// Load configuration
	if !a.noConfig {
		if err := a.loadConfig(cmd); err != nil {
			return err
		}
	}

	// Complete and validate options
	if a.options != nil {
		if err := a.options.Complete(); err != nil {
			return err
		}
		if err := a.options.Validate(); err != nil {
			return err
		}
	}

	// Run the application
	if a.runFunc != nil {
		return a.runFunc()
	}

	return nil
}

// loadConfig loads configuration from file, environment, and flags.
func (a *App) loadConfig(cmd *cobra.Command) error {
	// Get config file path
	configFile, _ := cmd.Flags().GetString("config")

	// Set config file
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		// Search for config file
		viper.SetConfigName(a.name)
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(filepath.Join(os.Getenv("HOME"), "."+a.name))
		viper.AddConfigPath("/etc/" + a.name)
	}

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, continue without it
	}

	// Expand environment variables in config values
	expandEnvVars()

	// Bind environment variables
	viper.SetEnvPrefix(strings.ToUpper(strings.ReplaceAll(a.name, "-", "_")))
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// Unmarshal config into options
	if a.options != nil {
		// Capture changed flags to preserve precedence
		changedFlags := make(map[string]string)
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Changed {
				changedFlags[f.Name] = f.Value.String()
			}
		})

		if err := viper.Unmarshal(a.options); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		// Re-apply changed flags
		for name, val := range changedFlags {
			if err := cmd.Flags().Set(name, val); err != nil {
				return fmt.Errorf("failed to re-apply flag %s: %w", name, err)
			}
		}
	}

	return nil
}

// expandEnvVars expands ${VAR} and $VAR style environment variables in config values.
func expandEnvVars() {
	// Pattern for ${VAR} or $VAR
	envPattern := regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

	for _, key := range viper.AllKeys() {
		val := viper.Get(key)
		if strVal, ok := val.(string); ok {
			expanded := envPattern.ReplaceAllStringFunc(strVal, func(match string) string {
				var varName string
				if strings.HasPrefix(match, "${") {
					varName = match[2 : len(match)-1]
				} else {
					varName = match[1:]
				}
				if envVal := os.Getenv(varName); envVal != "" {
					return envVal
				}
				return match // 保留原样，如果环境变量不存在
			})
			if expanded != strVal {
				viper.Set(key, expanded)
			}
		}
	}
}

// Run executes the application.
func (a *App) Run() {
	if err := a.cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Command returns the cobra command.
func (a *App) Command() *cobra.Command {
	return a.cmd
}
