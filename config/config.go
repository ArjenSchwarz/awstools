package config

// Config holds the global configuration settings
type Config struct {
	Verbose        *bool
	OutputFile     *string
	OutputFormat   *string
	OutputHeaders  *string
	AppendToOutput *bool
	NameFile       *string
	DotColumns     *DotColumns
}

// DotColumns is used to set the From and To columns for the dot output format
type DotColumns struct {
	From string
	To   string
}
