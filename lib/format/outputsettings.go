package format

import (
	"github.com/ArjenSchwarz/awstools/config"
	"github.com/ArjenSchwarz/awstools/drawio"
)

type OutputSettings struct {
	UseEmoji     bool
	OutputFormat string
	OutputFile   string
	ShouldAppend bool
	Title        string
	SortKey      string
	DrawIOHeader drawio.Header
	DotColumns   *DotColumns
}

// DotColumns is used to set the From and To columns for the dot output format
type DotColumns struct {
	From string
	To   string
}

func NewOutputSettings(config config.Config) *OutputSettings {
	settings := OutputSettings{
		UseEmoji:     *config.UseEmoji,
		OutputFormat: *config.OutputFormat,
		OutputFile:   *config.OutputFile,
		ShouldAppend: *config.AppendToOutput,
	}
	return &settings
}

func (settings *OutputSettings) AddDotFromToColumns(from string, to string) {
	dotcolumns := DotColumns{
		From: from,
		To:   to,
	}
	settings.DotColumns = &dotcolumns
}
