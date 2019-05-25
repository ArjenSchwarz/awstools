package config

// Config holds the global configuration settings
type Config struct {
	Verbose       *bool
	OutputFile    *string
	OutputFormat  *string
	OutputHeaders *string
}
