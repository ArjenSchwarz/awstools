package config

import (
	"testing"

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
		viper.Set("output.append", true)
		viper.Set("output.table.style", "default")
		viper.Set("output.table.max-column-width", 50)

		settings := config.NewOutputSettings()

		assert.True(t, settings.UseEmoji)
		assert.Equal(t, "json", settings.OutputFormat)
		assert.Equal(t, "/tmp/output.json", settings.OutputFile)
		assert.True(t, settings.ShouldAppend)
		assert.Equal(t, 50, settings.TableMaxColumnWidth)

		viper.Reset()
	})

	t.Run("creates output settings with defaults when not set", func(t *testing.T) {
		viper.Reset()

		settings := config.NewOutputSettings()

		assert.False(t, settings.UseEmoji)
		assert.Equal(t, "", settings.OutputFormat)
		assert.Equal(t, "", settings.OutputFile)
		assert.False(t, settings.ShouldAppend)
		assert.Equal(t, 0, settings.TableMaxColumnWidth)
	})
}
