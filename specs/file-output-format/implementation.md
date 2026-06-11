# Implementation Explanation: File Output Format (T-1535)

Branch `T-1535/file-output-format`, two commits implementing `specs/file-output-format/smolspec.md`.

## Beginner Level

### What Changed
awstools commands can print their results to the screen and, optionally, also save them to a file with `--file`. Until now the file always contained exactly what the screen showed: ask for a table, get a table in the file too. This change adds a new option, `--file-format`, that lets the file use a different format. You can now look at a readable table on screen while the saved file contains JSON that scripts can process.

### Why It Matters
People using awstools often want two things at once from a single run: something readable for themselves and something parseable for automation. Without this flag they had to run the same command twice, which means twice the AWS API calls — these are slow and sometimes rate-limited. The user's other tool, fog, already works this way; awstools now matches it.

### Key Concepts
- **Output format**: how results are presented — `json` (machine-readable), `table` (human-readable), `csv` (spreadsheets), and so on.
- **Flag**: an option you pass on the command line, like `--file out.json`. A *global* (persistent) flag works on every awstools subcommand.
- **Config file**: instead of typing flags every time, awstools reads defaults from `.awstools.yaml`. The new flag can be set there as `output.file-format`.

Example: `awstools s3 list -o table --file buckets.json --file-format json` shows a table and writes JSON.

## Intermediate Level

### Changes Overview
- `cmd/root.go`: new persistent flag `--file-format` (default empty) bound to viper key `output.file-format`; the stale `--output` help text was corrected (it omitted yaml, markdown, mermaid).
- `config/config.go`: `NewOutputSettings()` now populates `OutputSettings.OutputFileFormat` via `GetLCString` (lowercased, matching how `output.format` is normalised). `ShouldCombineAndAppend()` now evaluates its HTML exclusion against the *effective file format* (file-format when set, else the stdout format) and builds the settings object once instead of twice.
- `config/config_test.go`: subtests for set/unset/mixed-case values, divergent-format combine-and-append cases, and an end-to-end test that calls `OutputArray.Write()` with json stdout + csv file-format and asserts the file content is CSV.
- `cmd/root_flags_test.go`: asserts the flag is registered on the root command and bound to the viper key.
- `README.md`: help capture and format prose updated.

### Implementation Approach
The rendering split already existed: go-output v1.4.0's `OutputArray.Write()` renders stdout from `OutputFormat` and, when `OutputFile` is set, re-renders from `OutputFileFormat`, falling back to `OutputFormat` when empty. awstools simply never set that field. The whole feature is therefore wiring: one flag, one viper binding, one field assignment — following the exact flag/binding pattern already in `cmd/root.go`.

### Trade-offs
- **Stay on go-output v1 vs migrate to v2**: fog implements this on v2's multi-writer model. Migrating awstools would rewrite the output path of every command for no functional gain here (decision log #1).
- **No format validation**: `--output` doesn't validate values (unknown ones fall back to JSON in the library), so `--file-format` behaves identically rather than introducing an inconsistent failure mode (decision log #3).
- **`ShouldCombineAndAppend` fix vs documenting it as undefined**: keying the HTML exclusion to the file format is a few lines and prevents a foreseeable parse failure, so it was fixed rather than scoped out (decision log #2).

## Expert Level

### Technical Deep Dive
`Write()` in v1.4.0 is two sequential render passes over the same `OutputArray`. The stdout pass renders `OutputFormat` and prints via `PrintByteSlice(result, "", ...)`; the file pass mutates `Settings.OutputFileFormat` to `OutputFormat` when empty, then renders again and writes to `OutputFile`. Because awstools constructs a fresh `OutputSettings` per command invocation (`NewOutputSettings()`), the library's mutation of the settings object has no cross-command effect.

The `GetLCString` choice matters: `SetOutputFormat()` lowercases internally, but `OutputFileFormat` is a bare field with no setter, so the lowercasing must happen on the awstools side to keep `--file-format CSV` working and to keep the `ShouldCombineAndAppend` comparison (`!= "html"`) reliable.

`ShouldCombineAndAppend()` gates the multi-account collection flow (`cmd/vpcpeerings.go`, `cmd/tgwoverview.go`, `cmd/names.go`) that re-reads the existing output file via `drawio.GetHeaderAndContentsFromFile` and merges prior rows. The HTML exclusion exists because the HTML renderer handles appending itself. That check is semantically about the file, so with divergent formats it must consult the effective file format — otherwise `-o csv --file x.html --file-format html` would attempt to parse HTML as CSV rows.

### Architecture Impact
Zero new abstractions; one new field flows through the existing settings pipeline. Every command picks the feature up automatically because output handling is centralised in `Config.NewOutputSettings()`. A future go-output v2 migration maps this directly onto fog's `GetFileOutputOptions()` pattern (separate Output instance when formats differ).

### Potential Issues
- **v1 renderer quirks are inherited, not fixed**: the HTML renderer writes directly to `OutputSettings.OutputFile`, so `-o html --file x --file-format csv` writes HTML to the file during the stdout pass, then overwrites it with CSV in the file pass. Stdout shows nothing for the HTML pass. Pre-existing, documented out of scope.
- **Graph formats as file format**: `drawio`/`dot`/`mermaid` require per-command configuration (`DrawIOHeader`, `FromToColumns`) that commands only set when the *stdout* format requests them (`settings.IsDrawIO()` checks `OutputFormat`). Requesting them only via `--file-format` hits the library's `log.Fatal`. Same failure as an unsupported `--output` today.
- **Append with format drift**: nothing prevents appending csv rows to a file that previously contained json. Already possible by varying `--output` between runs; remains user responsibility.
- **Buffered multi-section commands**: `vpc enis` builds a combined `OutputArray` precisely because the library's section buffer is consumed by the stdout pass; the file pass re-renders from `Contents` only. That workaround keeps working with divergent formats.

## Completeness Assessment

**Fully implemented**: all six MUST requirements (global flag, fallback default, same value handling, config key support, stdout untouched, combine-and-append keyed to effective file format) and both SHOULD items (help text states the default; `--output` help corrected). Tests cover config wiring, combine-and-append divergence, flag registration/binding, and an end-to-end divergent-format write. CLI behaviour verified manually with `demo tables -o table --file out --file-format json`.

**Partially implemented**: none.

**Missing**: nothing against the spec. The out-of-scope list (v2 migration, renderer quirks, validation, docs/ regeneration) is intentional and recorded in the decision log.
