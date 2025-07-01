package config

import (
	"strings"

	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/viper"
)

// Config holds the global configuration settings
type Config struct {
}

// GetLCString returns a lowercase string value for the given setting
func (config *Config) GetLCString(setting string) string {
	if viper.IsSet(setting) {
		return strings.ToLower(viper.GetString(setting))
	}
	return ""
}

// GetOutputFormat returns the configured output format
func (config *Config) GetOutputFormat() string {
	return config.GetLCString("output.format")
}

// GetString returns a string value for the given setting
func (config *Config) GetString(setting string) string {
	if viper.IsSet(setting) {
		return viper.GetString(setting)
	}
	return ""
}

// GetBool returns a boolean value for the given setting
func (config *Config) GetBool(setting string) bool {
	return viper.GetBool(setting)
}

// GetInt returns an integer value for the given setting
func (config *Config) GetInt(setting string) int {
	if viper.IsSet(setting) {
		return viper.GetInt(setting)
	}
	return 0
}

// GetSeparator returns the appropriate separator string based on output format
func (config *Config) GetSeparator() string {
	switch config.NewOutputSettings().OutputFormat {
	case "table":
		return "\r\n"
	case "dot":
		return ","
	default:
		return ", "
	}
}

// IsDrawIO returns if output is set to Draw.IO
func (config *Config) IsDrawIO() bool {
	return config.NewOutputSettings().OutputFormat == "drawio"
}

// ShouldAppend returns if the output should append
func (config *Config) ShouldAppend() bool {
	return config.GetBool("output.append")
}

// ShouldCombineAndAppend returns if the output should be combined
func (config *Config) ShouldCombineAndAppend() bool {
	if !config.NewOutputSettings().ShouldAppend {
		return false
	}
	if config.NewOutputSettings().OutputFormat == "html" {
		return false
	}
	return true
}

// IsVerbose returns whether verbose output is enabled
func (config *Config) IsVerbose() bool {
	return config.GetBool("output.verbose")
}

// NewOutputSettings creates and returns a new OutputSettings instance with current configuration
func (config *Config) NewOutputSettings() *format.OutputSettings {
	settings := format.NewOutputSettings()
	settings.UseEmoji = config.GetBool("output.use-emoji")
	settings.SetOutputFormat(config.GetLCString("output.format"))
	settings.OutputFile = config.GetLCString("output.file")
	settings.ShouldAppend = config.GetBool("output.append")
	settings.TableStyle = format.TableStyles[config.GetString("output.table.style")]
	settings.TableMaxColumnWidth = config.GetInt("output.table.max-column-width")
	return settings
}
