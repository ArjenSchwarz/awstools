package config

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	output "github.com/ArjenSchwarz/go-output/v2"
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

// IsDrawIO returns whether either destination (stdout or the effective file
// format) renders Draw.IO, so commands know to build their DrawIO flavor.
func (config *Config) IsDrawIO() bool {
	return config.GetOutputFormat() == output.FormatDrawIO || config.effectiveFileFormat() == output.FormatDrawIO
}

// NeedsGraphFormat returns whether either destination (stdout or the effective
// file format) renders a graph format (dot/mermaid), so commands know to build
// their Graph flavor.
func (config *Config) NeedsGraphFormat() bool {
	return isGraphFormat(config.GetOutputFormat()) || isGraphFormat(config.effectiveFileFormat())
}

func isGraphFormat(name string) bool {
	return name == output.FormatDOT || name == output.FormatMermaid
}

// effectiveFileFormat returns the format the file destination is written in:
// output.file-format when set, otherwise the stdout format.
func (config *Config) effectiveFileFormat() string {
	if fileFormat := config.GetLCString("output.file-format"); fileFormat != "" {
		return fileFormat
	}
	return config.GetOutputFormat()
}

// ShouldAppend returns if the output should append
func (config *Config) ShouldAppend() bool {
	return config.GetBool("output.append")
}

// ShouldCombineAndAppend returns if the output should be combined
func (config *Config) ShouldCombineAndAppend() bool {
	if !config.ShouldAppend() {
		return false
	}
	// The HTML exclusion concerns the file being appended to, so it has to
	// look at the format the file is written in, not the stdout format.
	return config.effectiveFileFormat() != "html"
}

// IsVerbose returns whether verbose output is enabled
func (config *Config) IsVerbose() bool {
	return config.GetBool("output.verbose")
}

// DocumentSet holds per-format-family document flavors. Table is required;
// Graph and DrawIO stay nil unless the command supports those formats. A
// document handed to a format must contain only content appropriate for that
// format family, because v2 renderers walk all document contents.
type DocumentSet struct {
	Table  *output.Document // json/yaml/csv/table/markdown/html
	Graph  *output.Document // dot/mermaid
	DrawIO *output.Document // drawio
}

// flavorFor returns the document flavor matching the given format name. When a
// graph or drawio format is requested but the command did not populate that
// flavor, it returns the v1-compatible unsupported-format error.
func (docs DocumentSet) flavorFor(formatName string) (*output.Document, error) {
	var flavor *output.Document
	switch formatName {
	case output.FormatDOT, output.FormatMermaid:
		flavor = docs.Graph
	case output.FormatDrawIO:
		flavor = docs.DrawIO
	default:
		return docs.Table, nil
	}
	if flavor == nil {
		//nolint:staticcheck // ST1005: capitalized v1 message preserved verbatim (R9.2)
		return nil, fmt.Errorf("This command doesn't currently support the %s output format", formatName)
	}
	return flavor, nil
}

// RenderOption adjusts how RenderDocuments writes its destinations.
type RenderOption func(*renderOptions)

type renderOptions struct {
	fileOverwrite bool
}

// WithFileOverwrite writes the file destination fresh even when output.append
// is set. Combine-style commands (tgw overview, vpc peerings) pass this after
// merging the prior file contents into the document themselves, so appending
// would duplicate the merged rows.
func WithFileOverwrite() RenderOption {
	return func(options *renderOptions) {
		options.fileOverwrite = true
	}
}

// RenderDocuments renders the flavor matching each destination's format:
// stdout in output.format, then output.file in the effective file format when
// a file is set. It returns the first error and never exits the process.
func (config *Config) RenderDocuments(ctx context.Context, docs DocumentSet, opts ...RenderOption) error {
	return config.renderDocuments(ctx, docs, output.NewStdoutWriter(), opts...)
}

// RenderDocument is sugar for table-only commands:
// RenderDocuments(ctx, DocumentSet{Table: doc}).
func (config *Config) RenderDocument(ctx context.Context, doc *output.Document) error {
	return config.RenderDocuments(ctx, DocumentSet{Table: doc})
}

// renderDocuments is the render core with an injectable stdout writer so tests
// can capture the stdout bytes without redirecting os.Stdout.
func (config *Config) renderDocuments(ctx context.Context, docs DocumentSet, stdout output.Writer, opts ...RenderOption) error {
	options := renderOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	stdoutFormat := config.GetOutputFormat()
	stdoutDoc, err := docs.flavorFor(stdoutFormat)
	if err != nil {
		return err
	}
	stdoutOutput := output.NewOutput(config.outputOptions(stdoutFormat, stdout)...)
	if err := stdoutOutput.Render(ctx, stdoutDoc); err != nil {
		return err
	}

	file := config.GetString("output.file")
	if file == "" {
		return nil
	}
	fileFormat := config.effectiveFileFormat()
	fileDoc, err := docs.flavorFor(fileFormat)
	if err != nil {
		return err
	}
	var writerOpts []output.FileWriterOption
	if config.ShouldAppend() && !options.fileOverwrite {
		writerOpts = append(writerOpts, output.WithAppendMode())
	}
	// The pattern is the exact base name (no {format}/{ext} placeholders), so
	// the configured filename is preserved as-is.
	fileWriter, err := output.NewFileWriterWithOptions(filepath.Dir(file), filepath.Base(file), writerOpts...)
	if err != nil {
		return err
	}
	fileOutput := output.NewOutput(config.outputOptions(fileFormat, fileWriter)...)
	return fileOutput.Render(ctx, fileDoc)
}

// outputOptions assembles the Output options shared by both destinations:
// the format for the given name, the destination writer, and the emoji
// transformer when output.use-emoji is set.
func (config *Config) outputOptions(formatName string, writer output.Writer) []output.OutputOption {
	opts := []output.OutputOption{
		output.WithFormat(formatFor(formatName, config)),
		output.WithWriter(writer),
	}
	if config.GetBool("output.use-emoji") {
		opts = append(opts, output.WithTransformer(&output.EmojiTransformer{}))
	}
	return opts
}

// formatFor maps a config format name to its v2 Format. Unknown and empty
// values fall back to JSON, matching v1's no-validation behavior for both
// --output and --file-format (R3.1, R3.6).
func formatFor(name string, config *Config) output.Format {
	switch name {
	case "yaml":
		return output.YAML()
	case "csv":
		return output.CSV()
	case "table":
		return output.TableWithStyleAndMaxColumnWidth(
			config.GetString("output.table.style"),
			config.GetInt("output.table.max-column-width"),
		)
	case "markdown":
		return output.Markdown()
	case "html":
		return output.HTMLWithTemplate(output.DefaultHTMLTemplate)
	case output.FormatDOT:
		return output.DOT()
	case output.FormatMermaid:
		return output.Mermaid()
	case output.FormatDrawIO:
		return output.DrawIO()
	default:
		return output.JSON()
	}
}

// SortOption returns a TableOption sorting the table by column, ascending —
// the v1 SortKey equivalent (R3.4).
func SortOption(column string) output.TableOption {
	return output.WithTransformations(output.NewSortOp(output.SortKey{Column: column, Direction: output.Ascending}))
}
