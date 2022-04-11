package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds the global configuration settings
type Config struct {
	Verbose        *bool
	OutputFile     *string
	OutputFormat   *string
	AppendToOutput *bool
	NameFile       *string
	Profile        *string
	Region         *string
	UseEmoji       *bool
}

func (config *Config) GetLCString(setting string) string {
	if viper.IsSet(setting) {
		return strings.ToLower(viper.GetString(setting))
	}
	return ""
}

// GetOutputFormat returns the output format
func (config *Config) GetOutputFormat() string {
	return strings.ToLower(*config.OutputFormat)
}

// IsDrawIO returns if output is set to Draw.IO
func (config *Config) IsDrawIO() bool {
	return config.GetOutputFormat() == "drawio"
}

// ShouldAppend returns if the output should append
func (config *Config) ShouldAppend() bool {
	return *config.AppendToOutput
}

// ShouldCombineAndAppend returns if the output should be combined
func (config *Config) ShouldCombineAndAppend() bool {
	if !config.ShouldAppend() {
		return false
	}
	if config.GetOutputFormat() == "html" {
		return false
	}
	return true
}
