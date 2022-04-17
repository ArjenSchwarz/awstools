package format

import (
	"strings"

	"github.com/ArjenSchwarz/awstools/drawio"
	"github.com/jedib0t/go-pretty/v6/table"
)

// TableStyles is a lookup map for getting the table styles based on a string
var TableStyles = map[string]table.Style{
	"Default":                    table.StyleDefault,
	"Bold":                       table.StyleBold,
	"ColoredBright":              table.StyleColoredBright,
	"ColoredDark":                table.StyleColoredDark,
	"ColoredBlackOnBlueWhite":    table.StyleColoredBlackOnBlueWhite,
	"ColoredBlackOnCyanWhite":    table.StyleColoredBlackOnCyanWhite,
	"ColoredBlackOnGreenWhite":   table.StyleColoredBlackOnGreenWhite,
	"ColoredBlackOnMagentaWhite": table.StyleColoredBlackOnMagentaWhite,
	"ColoredBlackOnYellowWhite":  table.StyleColoredBlackOnYellowWhite,
	"ColoredBlackOnRedWhite":     table.StyleColoredBlackOnRedWhite,
	"ColoredBlueWhiteOnBlack":    table.StyleColoredBlueWhiteOnBlack,
	"ColoredCyanWhiteOnBlack":    table.StyleColoredCyanWhiteOnBlack,
	"ColoredGreenWhiteOnBlack":   table.StyleColoredGreenWhiteOnBlack,
	"ColoredMagentaWhiteOnBlack": table.StyleColoredMagentaWhiteOnBlack,
	"ColoredRedWhiteOnBlack":     table.StyleColoredRedWhiteOnBlack,
	"ColoredYellowWhiteOnBlack":  table.StyleColoredYellowWhiteOnBlack,
}

type OutputSettings struct {
	UseEmoji            bool
	OutputFormat        string
	OutputFile          string
	ShouldAppend        bool
	Title               string
	SortKey             string
	DrawIOHeader        drawio.Header
	FromToColumns       *FromToColumns
	TableStyle          table.Style
	SeparateTables      bool
	TableMaxColumnWidth int
}

// DotColumns is used to set the From and To columns for the dot output format
type FromToColumns struct {
	From string
	To   string
}

func NewOutputSettings() *OutputSettings {
	settings := OutputSettings{
		TableStyle:          table.StyleDefault,
		TableMaxColumnWidth: 50,
	}
	return &settings
}

func (settings *OutputSettings) AddFromToColumns(from string, to string) {
	result := FromToColumns{
		From: from,
		To:   to,
	}
	settings.FromToColumns = &result
}

func (settings *OutputSettings) SetOutputFormat(format string) {
	settings.OutputFormat = strings.ToLower(format)
}

func (settings *OutputSettings) NeedsFromToColumns() bool {
	if settings.OutputFormat == "dot" || settings.OutputFormat == "mermaid" {
		return true
	}
	return false
}
