package config

import (
	"strings"

	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/viper"
)

// Config holds the global configuration settings
type Config struct {
}

func (config *Config) GetLCString(setting string) string {
	if viper.IsSet(setting) {
		return strings.ToLower(viper.GetString(setting))
	}
	return ""
}

func (config *Config) GetOutputFormat() string {
	return config.GetLCString("output.format")
}

func (config *Config) GetString(setting string) string {
	if viper.IsSet(setting) {
		return viper.GetString(setting)
	}
	return ""
}

func (config *Config) GetBool(setting string) bool {
	return viper.GetBool(setting)
}

func (config *Config) GetInt(setting string) int {
	if viper.IsSet(setting) {
		return viper.GetInt(setting)
	}
	return 0
}

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

func (config *Config) IsVerbose() bool {
	return config.GetBool("output.verbose")
}

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
