package format

import (
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
	DotColumns          *DotColumns
	TableStyle          table.Style
	SeparateTables      bool
	TableMaxColumnWidth int
}

// DotColumns is used to set the From and To columns for the dot output format
type DotColumns struct {
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

func (settings *OutputSettings) AddDotFromToColumns(from string, to string) {
	dotcolumns := DotColumns{
		From: from,
		To:   to,
	}
	settings.DotColumns = &dotcolumns
}
