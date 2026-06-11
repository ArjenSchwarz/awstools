package config

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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
