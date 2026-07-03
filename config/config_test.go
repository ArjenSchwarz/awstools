package config

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	format "github.com/ArjenSchwarz/go-output"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_GetLCString(t *testing.T) {
	config := &Config{}

	t.Run("returns lowercase string when setting exists", func(t *testing.T) {
		viper.Set("test.setting", "UPPERCASE")
		result := config.GetLCString("test.setting")
		assert.Equal(t, "uppercase", result)
		viper.Reset()
	})

	t.Run("returns empty string when setting does not exist", func(t *testing.T) {
		viper.Reset()
		result := config.GetLCString("nonexistent.setting")
		assert.Equal(t, "", result)
	})
}

func TestConfig_GetOutputFormat(t *testing.T) {
	config := &Config{}

	t.Run("returns output format when set", func(t *testing.T) {
		viper.Set("output.format", "JSON")
		result := config.GetOutputFormat()
		assert.Equal(t, "json", result)
		viper.Reset()
	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		viper.Reset()
		result := config.GetOutputFormat()
		assert.Equal(t, "", result)
	})
}

func TestConfig_GetString(t *testing.T) {
	config := &Config{}

	t.Run("returns string value when setting exists", func(t *testing.T) {
		viper.Set("test.setting", "testvalue")
		result := config.GetString("test.setting")
		assert.Equal(t, "testvalue", result)
		viper.Reset()
	})

	t.Run("returns empty string when setting does not exist", func(t *testing.T) {
		viper.Reset()
		result := config.GetString("nonexistent.setting")
		assert.Equal(t, "", result)
	})
}

func TestConfig_GetBool(t *testing.T) {
	config := &Config{}

	t.Run("returns true when setting is true", func(t *testing.T) {
		viper.Set("test.bool", true)
		result := config.GetBool("test.bool")
		assert.True(t, result)
		viper.Reset()
	})

	t.Run("returns false when setting is false", func(t *testing.T) {
		viper.Set("test.bool", false)
		result := config.GetBool("test.bool")
		assert.False(t, result)
		viper.Reset()
	})

	t.Run("returns false when setting does not exist", func(t *testing.T) {
		viper.Reset()
		result := config.GetBool("nonexistent.bool")
		assert.False(t, result)
	})
}

func TestConfig_GetInt(t *testing.T) {
	config := &Config{}

	t.Run("returns integer value when setting exists", func(t *testing.T) {
		viper.Set("test.int", 42)
		result := config.GetInt("test.int")
		assert.Equal(t, 42, result)
		viper.Reset()
	})

	t.Run("returns zero when setting does not exist", func(t *testing.T) {
		viper.Reset()
		result := config.GetInt("nonexistent.int")
		assert.Equal(t, 0, result)
	})
}

func TestConfig_GetSeparator(t *testing.T) {
	config := &Config{}

	t.Run("returns newline for table format", func(t *testing.T) {
		viper.Set("output.format", "table")
		result := config.GetSeparator()
		assert.Equal(t, "\r\n", result)
		viper.Reset()
	})

	t.Run("returns comma for dot format", func(t *testing.T) {
		viper.Set("output.format", "dot")
		result := config.GetSeparator()
		assert.Equal(t, ",", result)
		viper.Reset()
	})

	t.Run("returns comma space for other formats", func(t *testing.T) {
		viper.Set("output.format", "json")
		result := config.GetSeparator()
		assert.Equal(t, ", ", result)
		viper.Reset()
	})

	t.Run("returns comma space for empty format", func(t *testing.T) {
		viper.Reset()
		result := config.GetSeparator()
		assert.Equal(t, ", ", result)
	})
}

func TestConfig_IsDrawIO(t *testing.T) {
	config := &Config{}

	t.Run("returns true when format is drawio", func(t *testing.T) {
		viper.Set("output.format", "drawio")
		result := config.IsDrawIO()
		assert.True(t, result)
		viper.Reset()
	})

	t.Run("returns false when format is not drawio", func(t *testing.T) {
		viper.Set("output.format", "json")
		result := config.IsDrawIO()
		assert.False(t, result)
		viper.Reset()
	})

	t.Run("returns true when the file format is drawio", func(t *testing.T) {
		viper.Set("output.format", "json")
		viper.Set("output.file-format", "drawio")
		result := config.IsDrawIO()
		assert.True(t, result)
		viper.Reset()
	})

	t.Run("returns true when stdout is drawio and the file format is not", func(t *testing.T) {
		viper.Set("output.format", "drawio")
		viper.Set("output.file-format", "json")
		result := config.IsDrawIO()
		assert.True(t, result)
		viper.Reset()
	})
}

func TestConfig_ShouldAppend(t *testing.T) {
	config := &Config{}

	t.Run("returns true when append is enabled", func(t *testing.T) {
		viper.Set("output.append", true)
		result := config.ShouldAppend()
		assert.True(t, result)
		viper.Reset()
	})

	t.Run("returns false when append is disabled", func(t *testing.T) {
		viper.Set("output.append", false)
		result := config.ShouldAppend()
		assert.False(t, result)
		viper.Reset()
	})
}

func TestConfig_ShouldCombineAndAppend(t *testing.T) {
	config := &Config{}

	t.Run("returns false when append is disabled", func(t *testing.T) {
		viper.Set("output.append", false)
		viper.Set("output.format", "json")
		result := config.ShouldCombineAndAppend()
		assert.False(t, result)
		viper.Reset()
	})

	t.Run("returns false when format is html", func(t *testing.T) {
		viper.Set("output.append", true)
		viper.Set("output.format", "html")
		result := config.ShouldCombineAndAppend()
		assert.False(t, result)
		viper.Reset()
	})

	t.Run("returns true when append is enabled and format is not html", func(t *testing.T) {
		viper.Set("output.append", true)
		viper.Set("output.format", "json")
		result := config.ShouldCombineAndAppend()
		assert.True(t, result)
		viper.Reset()
	})

	t.Run("returns false when the file format is html even though stdout format is not", func(t *testing.T) {
		viper.Set("output.append", true)
		viper.Set("output.format", "csv")
		viper.Set("output.file-format", "html")
		result := config.ShouldCombineAndAppend()
		assert.False(t, result)
		viper.Reset()
	})

	t.Run("returns true when the file format overrides an html stdout format", func(t *testing.T) {
		viper.Set("output.append", true)
		viper.Set("output.format", "html")
		viper.Set("output.file-format", "csv")
		result := config.ShouldCombineAndAppend()
		assert.True(t, result)
		viper.Reset()
	})
}

func TestConfig_IsVerbose(t *testing.T) {
	config := &Config{}

	t.Run("returns true when verbose is enabled", func(t *testing.T) {
		viper.Set("output.verbose", true)
		result := config.IsVerbose()
		assert.True(t, result)
		viper.Reset()
	})

	t.Run("returns false when verbose is disabled", func(t *testing.T) {
		viper.Set("output.verbose", false)
		result := config.IsVerbose()
		assert.False(t, result)
		viper.Reset()
	})
}

func TestConfig_NewOutputSettings(t *testing.T) {
	config := &Config{}

	t.Run("creates output settings with correct values", func(t *testing.T) {
		viper.Set("output.use-emoji", true)
		viper.Set("output.format", "json")
		viper.Set("output.file", "/tmp/output.json")
		viper.Set("output.file-format", "csv")
		viper.Set("output.append", true)
		viper.Set("output.table.style", "default")
		viper.Set("output.table.max-column-width", 50)

		settings := config.NewOutputSettings()

		assert.True(t, settings.UseEmoji)
		assert.Equal(t, "json", settings.OutputFormat)
		assert.Equal(t, "/tmp/output.json", settings.OutputFile)
		assert.Equal(t, "csv", settings.OutputFileFormat)
		assert.True(t, settings.ShouldAppend)
		assert.Equal(t, 50, settings.TableMaxColumnWidth)

		viper.Reset()
	})

	t.Run("lowercases the file format like the output format", func(t *testing.T) {
		viper.Set("output.file-format", "CSV")

		settings := config.NewOutputSettings()

		assert.Equal(t, "csv", settings.OutputFileFormat)
		viper.Reset()
	})

	t.Run("preserves case of output file path", func(t *testing.T) {
		// Regression test for T-406: GetLCString lowercased file paths,
		// breaking paths with uppercase characters on case-sensitive filesystems.
		viper.Set("output.file", "/tmp/MyProject/Output.json")

		settings := config.NewOutputSettings()

		assert.Equal(t, "/tmp/MyProject/Output.json", settings.OutputFile)
		viper.Reset()
	})

	t.Run("creates output settings with defaults when not set", func(t *testing.T) {
		viper.Reset()

		settings := config.NewOutputSettings()

		assert.False(t, settings.UseEmoji)
		assert.Equal(t, "", settings.OutputFormat)
		assert.Equal(t, "", settings.OutputFile)
		// An empty OutputFileFormat makes the library fall back to the
		// stdout format when writing the file.
		assert.Equal(t, "", settings.OutputFileFormat)
		assert.False(t, settings.ShouldAppend)
		assert.Equal(t, 0, settings.TableMaxColumnWidth)
	})
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what
// was written. Write() prints the stdout rendering via os.Stdout directly, so
// this is the only way to observe it in a test.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}
	os.Stdout = old
	captured, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	return string(captured)
}

func TestConfig_FileFormatWrite(t *testing.T) {
	config := &Config{}

	t.Run("writes the file in the file format while stdout keeps the output format", func(t *testing.T) {
		outputFile := filepath.Join(t.TempDir(), "output")
		viper.Set("output.format", "json")
		viper.Set("output.file", outputFile)
		viper.Set("output.file-format", "csv")
		t.Cleanup(viper.Reset)

		stdout := captureStdout(t, func() {
			output := format.OutputArray{Keys: []string{"Name", "Value"}, Settings: config.NewOutputSettings()}
			output.AddContents(map[string]any{"Name": "first", "Value": "one"})
			output.Write()
		})

		contents, err := os.ReadFile(outputFile)
		assert.NoError(t, err)
		assert.Contains(t, string(contents), "Name,Value")
		assert.Contains(t, string(contents), "first,one")
		assert.NotContains(t, string(contents), "{")

		assert.Contains(t, stdout, `"Name":"first"`)
		assert.NotContains(t, stdout, "Name,Value")
	})

	t.Run("file format without a file is a no-op", func(t *testing.T) {
		outputDir := t.TempDir()
		viper.Set("output.format", "json")
		viper.Set("output.file-format", "csv")
		t.Cleanup(viper.Reset)

		stdout := captureStdout(t, func() {
			output := format.OutputArray{Keys: []string{"Name", "Value"}, Settings: config.NewOutputSettings()}
			output.AddContents(map[string]any{"Name": "first", "Value": "one"})
			output.Write()
		})

		assert.Contains(t, stdout, `"Name":"first"`)
		entries, err := os.ReadDir(outputDir)
		assert.NoError(t, err)
		assert.Empty(t, entries, "no file may be written when --file is not set")
	})
}

// ---- go-output v2 render helper tests ----

// captureWriter implements output.Writer, buffering rendered bytes so tests
// can inject it as the stdout writer of the renderDocuments core.
type captureWriter struct {
	buf bytes.Buffer
}

func (w *captureWriter) Write(_ context.Context, _ string, data []byte) error {
	_, err := w.buf.Write(data)
	return err
}

// tableDocument builds a single-table document with fixed keys for render tests.
func tableDocument(rows ...map[string]any) *output.Document {
	return output.New().Table("Test", rows, output.WithKeys("Name", "Value")).Build()
}

func TestFormatFor(t *testing.T) {
	config := &Config{}

	tests := map[string]struct {
		name string
		want string
	}{
		"json maps to json":                {name: "json", want: "json"},
		"yaml maps to yaml":                {name: "yaml", want: "yaml"},
		"csv maps to csv":                  {name: "csv", want: "csv"},
		"table maps to table":              {name: "table", want: "table"},
		"markdown maps to markdown":        {name: "markdown", want: "markdown"},
		"html maps to html":                {name: "html", want: "html"},
		"dot maps to dot":                  {name: "dot", want: "dot"},
		"mermaid maps to mermaid":          {name: "mermaid", want: "mermaid"},
		"drawio maps to drawio":            {name: "drawio", want: "drawio"},
		"empty falls back to json":         {name: "", want: "json"},
		"unknown value falls back to json": {name: "nosuchformat", want: "json"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := formatFor(tc.name, config)
			assert.Equal(t, tc.want, got.Name)
			assert.NotNil(t, got.Renderer)
		})
	}
}

func TestConfig_NeedsGraphFormat(t *testing.T) {
	config := &Config{}

	tests := map[string]struct {
		format     string
		fileFormat string
		want       bool
	}{
		"dot stdout":                        {format: "dot", want: true},
		"mermaid stdout":                    {format: "mermaid", want: true},
		"json stdout":                       {format: "json", want: false},
		"dot as file format":                {format: "json", fileFormat: "dot", want: true},
		"mermaid as file format":            {format: "json", fileFormat: "mermaid", want: true},
		"dot stdout with json file format":  {format: "dot", fileFormat: "json", want: true},
		"drawio is not a graph format":      {format: "drawio", want: false},
		"unset formats":                     {want: false},
		"non-graph stdout and file formats": {format: "json", fileFormat: "csv", want: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.format != "" {
				viper.Set("output.format", tc.format)
			}
			if tc.fileFormat != "" {
				viper.Set("output.file-format", tc.fileFormat)
			}
			t.Cleanup(viper.Reset)
			assert.Equal(t, tc.want, config.NeedsGraphFormat())
		})
	}
}

func TestConfig_RenderDocuments_FileFormat(t *testing.T) {
	config := &Config{}
	doc := tableDocument(map[string]any{"Name": "first", "Value": "one"})

	t.Run("writes the file in the file format while stdout keeps the output format", func(t *testing.T) {
		outputFile := filepath.Join(t.TempDir(), "out.csv")
		viper.Set("output.format", "json")
		viper.Set("output.file", outputFile)
		viper.Set("output.file-format", "csv")
		t.Cleanup(viper.Reset)

		stdout := &captureWriter{}
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout))

		// stdout parses as JSON and carries the data
		var stdoutData any
		require.NoError(t, json.Unmarshal(stdout.buf.Bytes(), &stdoutData), "stdout must be valid JSON")
		assert.Contains(t, stdout.buf.String(), "first")
		assert.NotContains(t, stdout.buf.String(), "Name,Value")

		// file parses as CSV with the expected column order and data
		contents, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.NotContains(t, string(contents), "{", "file must not contain JSON")
		records, err := csv.NewReader(bytes.NewReader(contents)).ReadAll()
		require.NoError(t, err, "file must be valid CSV")
		require.GreaterOrEqual(t, len(records), 2)
		assert.Equal(t, []string{"Name", "Value"}, records[0])
		assert.Equal(t, []string{"first", "one"}, records[1])
	})

	t.Run("file bytes are identical to stdout bytes when the formats match", func(t *testing.T) {
		outputFile := filepath.Join(t.TempDir(), "out.csv")
		viper.Set("output.format", "csv")
		viper.Set("output.file", outputFile)
		t.Cleanup(viper.Reset)

		stdout := &captureWriter{}
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout))

		contents, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.Equal(t, stdout.buf.Bytes(), contents)
	})

	t.Run("unknown file format falls back to json like an unknown output format", func(t *testing.T) {
		outputFile := filepath.Join(t.TempDir(), "out")
		viper.Set("output.format", "nosuchformat")
		viper.Set("output.file", outputFile)
		viper.Set("output.file-format", "alsonosuchformat")
		t.Cleanup(viper.Reset)

		stdout := &captureWriter{}
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout))

		var parsed any
		require.NoError(t, json.Unmarshal(stdout.buf.Bytes(), &parsed), "stdout must fall back to JSON")

		contents, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(contents, &parsed), "file must fall back to JSON")
	})

	t.Run("no file is written when output.file is not set", func(t *testing.T) {
		outputDir := t.TempDir()
		viper.Set("output.format", "json")
		viper.Set("output.file-format", "csv")
		t.Cleanup(viper.Reset)

		stdout := &captureWriter{}
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout))

		assert.Contains(t, stdout.buf.String(), "first")
		entries, err := os.ReadDir(outputDir)
		require.NoError(t, err)
		assert.Empty(t, entries, "no file may be written when --file is not set")
	})
}

func TestConfig_RenderDocuments_GraphFormatGuard(t *testing.T) {
	config := &Config{}
	doc := tableDocument(map[string]any{"Name": "first", "Value": "one"})

	for _, formatName := range []string{"dot", "mermaid", "drawio"} {
		t.Run(formatName+" without matching flavor errors with the v1 message", func(t *testing.T) {
			viper.Set("output.format", formatName)
			t.Cleanup(viper.Reset)

			stdout := &captureWriter{}
			err := config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout)
			require.Error(t, err)
			assert.Equal(t, "This command doesn't currently support the "+formatName+" output format", err.Error())
		})
	}

	t.Run("guard also fires for the file destination", func(t *testing.T) {
		outputFile := filepath.Join(t.TempDir(), "out.dot")
		viper.Set("output.format", "json")
		viper.Set("output.file", outputFile)
		viper.Set("output.file-format", "dot")
		t.Cleanup(viper.Reset)

		stdout := &captureWriter{}
		err := config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout)
		require.Error(t, err)
		assert.Equal(t, "This command doesn't currently support the dot output format", err.Error())
	})

	t.Run("graph flavor renders when populated", func(t *testing.T) {
		viper.Set("output.format", "dot")
		t.Cleanup(viper.Reset)

		docs := DocumentSet{
			Table: doc,
			Graph: output.New().Graph("Test", []output.Edge{{From: "node-a", To: "node-b"}}).Build(),
		}
		stdout := &captureWriter{}
		require.NoError(t, config.renderDocuments(t.Context(), docs, stdout))
		assert.Contains(t, stdout.buf.String(), "node-a")
		assert.Contains(t, stdout.buf.String(), "node-b")
	})

	t.Run("drawio flavor renders when populated", func(t *testing.T) {
		viper.Set("output.format", "drawio")
		t.Cleanup(viper.Reset)

		docs := DocumentSet{
			Table:  doc,
			DrawIO: output.New().DrawIO("Test", []output.Record{{"Name": "node-a"}}, output.DefaultDrawIOHeader()).Build(),
		}
		stdout := &captureWriter{}
		require.NoError(t, config.renderDocuments(t.Context(), docs, stdout))
		assert.Contains(t, stdout.buf.String(), "node-a")
	})
}

func TestConfig_RenderDocuments_EmojiTransformer(t *testing.T) {
	config := &Config{}
	doc := tableDocument(map[string]any{"Name": "first", "Value": "Yes"})

	t.Run("emoji substitution applies when output.use-emoji is set", func(t *testing.T) {
		viper.Set("output.format", "csv")
		viper.Set("output.use-emoji", true)
		t.Cleanup(viper.Reset)

		stdout := &captureWriter{}
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout))
		assert.Contains(t, stdout.buf.String(), "✅")
		assert.NotContains(t, stdout.buf.String(), "Yes")
	})

	t.Run("no emoji substitution when output.use-emoji is not set", func(t *testing.T) {
		viper.Set("output.format", "csv")
		t.Cleanup(viper.Reset)

		stdout := &captureWriter{}
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout))
		assert.Contains(t, stdout.buf.String(), "Yes")
		assert.NotContains(t, stdout.buf.String(), "✅")
	})

	t.Run("emoji substitution applies to the file output too", func(t *testing.T) {
		outputFile := filepath.Join(t.TempDir(), "out.csv")
		viper.Set("output.format", "json")
		viper.Set("output.file", outputFile)
		viper.Set("output.file-format", "csv")
		viper.Set("output.use-emoji", true)
		t.Cleanup(viper.Reset)

		stdout := &captureWriter{}
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout))

		contents, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.Contains(t, string(contents), "✅")
	})
}

func TestConfig_RenderDocuments_TableStyleAndWidth(t *testing.T) {
	config := &Config{}
	longValue := strings.Repeat("x", 40)
	doc := tableDocument(map[string]any{"Name": "first", "Value": longValue})

	t.Run("applies the configured table style", func(t *testing.T) {
		viper.Set("output.format", "table")
		viper.Set("output.table.style", "Rounded")
		t.Cleanup(viper.Reset)

		stdout := &captureWriter{}
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout))
		assert.Contains(t, stdout.buf.String(), "╭", "Rounded style must draw rounded corners")
	})

	t.Run("caps column width at output.table.max-column-width", func(t *testing.T) {
		viper.Set("output.format", "table")
		viper.Set("output.table.max-column-width", 10)
		t.Cleanup(viper.Reset)

		stdout := &captureWriter{}
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout))
		assert.NotContains(t, stdout.buf.String(), longValue, "long cells must wrap at the configured width")
	})

	t.Run("width zero leaves column width unlimited", func(t *testing.T) {
		viper.Set("output.format", "table")
		t.Cleanup(viper.Reset)

		stdout := &captureWriter{}
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout))
		assert.Contains(t, stdout.buf.String(), longValue)
	})
}

func TestConfig_RenderDocuments_AppendMode(t *testing.T) {
	config := &Config{}
	docFor := func(name string) *output.Document {
		return tableDocument(map[string]any{"Name": name, "Value": "one"})
	}

	t.Run("appends to the existing file when output.append is set", func(t *testing.T) {
		outputFile := filepath.Join(t.TempDir(), "out.csv")
		viper.Set("output.format", "csv")
		viper.Set("output.file", outputFile)
		viper.Set("output.append", true)
		t.Cleanup(viper.Reset)

		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: docFor("first")}, &captureWriter{}))
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: docFor("second")}, &captureWriter{}))

		contents, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.Contains(t, string(contents), "first")
		assert.Contains(t, string(contents), "second")
	})

	t.Run("WithFileOverwrite suppresses append mode", func(t *testing.T) {
		outputFile := filepath.Join(t.TempDir(), "out.csv")
		viper.Set("output.format", "csv")
		viper.Set("output.file", outputFile)
		viper.Set("output.append", true)
		t.Cleanup(viper.Reset)

		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: docFor("first")}, &captureWriter{}))
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: docFor("second")}, &captureWriter{}, WithFileOverwrite()))

		contents, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.NotContains(t, string(contents), "first")
		assert.Contains(t, string(contents), "second")
	})

	t.Run("overwrites by default when output.append is not set", func(t *testing.T) {
		outputFile := filepath.Join(t.TempDir(), "out.csv")
		viper.Set("output.format", "csv")
		viper.Set("output.file", outputFile)
		t.Cleanup(viper.Reset)

		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: docFor("first")}, &captureWriter{}))
		require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: docFor("second")}, &captureWriter{}))

		contents, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.NotContains(t, string(contents), "first")
		assert.Contains(t, string(contents), "second")
	})
}

func TestSortOption(t *testing.T) {
	config := &Config{}
	viper.Set("output.format", "csv")
	t.Cleanup(viper.Reset)

	rows := []map[string]any{
		{"Name": "zebra", "Value": "one"},
		{"Name": "apple", "Value": "two"},
	}
	doc := output.New().Table("Test", rows, output.WithKeys("Name", "Value"), SortOption("Name")).Build()

	stdout := &captureWriter{}
	require.NoError(t, config.renderDocuments(t.Context(), DocumentSet{Table: doc}, stdout))

	rendered := stdout.buf.String()
	assert.Less(t, strings.Index(rendered, "apple"), strings.Index(rendered, "zebra"),
		"rows must be sorted ascending by the sort column")
}

func TestConfig_RenderDocument(t *testing.T) {
	config := &Config{}
	viper.Set("output.format", "json")
	t.Cleanup(viper.Reset)

	doc := tableDocument(map[string]any{"Name": "first", "Value": "one"})

	stdout := captureStdout(t, func() {
		require.NoError(t, config.RenderDocument(t.Context(), doc))
	})
	assert.Contains(t, stdout, "first")
}
