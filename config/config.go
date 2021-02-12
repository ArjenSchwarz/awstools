package config

import (
	"strings"
)

// Config holds the global configuration settings
type Config struct {
	Verbose        *bool
	OutputFile     *string
	OutputFormat   *string
	AppendToOutput *bool
	NameFile       *string
	DotColumns     *DotColumns
}

// DotColumns is used to set the From and To columns for the dot output format
type DotColumns struct {
	From string
	To   string
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
