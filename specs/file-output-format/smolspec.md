# File Output Format

Transit ticket: T-1535

## Overview

awstools can write command output to a file with `--file`, but the file always uses the same format as stdout (`--output`). Users who want, for example, a human-readable table on screen and a machine-readable JSON file have to run the command twice. This change adds a `--file-format` flag, mirroring the equivalent flag in fog, so the file can use a different format than stdout.

## Requirements

- The system MUST accept a `--file-format` flag as a global persistent flag on the root command (available to every subcommand, exactly like `--file`) that specifies the output format used for the file written via `--file`.
- When `--file-format` is not provided, the system MUST write the file in the same format as stdout (current behaviour, unchanged).
- The system MUST accept the same format values for `--file-format` as for `--output`, with the same case-insensitive handling and the same fallback for unknown values (the library falls back to JSON; neither flag validates input).
- The system MUST support setting the file format via the config file (key `output.file-format` in `.awstools.yaml`), consistent with how `output.format` and `output.file` work.
- The system MUST NOT change stdout output in any way when `--file-format` is provided.
- The combine-and-append flow (`Config.ShouldCombineAndAppend`, used by multi-account collection commands) MUST evaluate its HTML exclusion against the effective file format (`--file-format` when set, otherwise `--output`), since that check is about the file being appended to.
- The flag's help text SHOULD state that it defaults to the value of `--output`.
- The `--output` flag's stale help text SHOULD be corrected while editing the adjacent line: it currently omits markdown, mermaid, and yaml, which the library supports.

## Implementation Approach

go-output v1.4.0 (already the pinned version in `go.mod`) natively supports this: `OutputSettings.OutputFileFormat` exists and `OutputArray.Write()` renders stdout using `OutputFormat` and the file using `OutputFileFormat`, falling back to `OutputFormat` when unset. awstools just never sets the field. The change is wiring only:

- `cmd/root.go`: add a persistent string flag `--file-format` (default `""`) alongside the existing `--file` flag, and bind it to viper key `output.file-format`, following the exact pattern of the existing `output.file` binding (`cmd/root.go:44-66`). Correct the `--output` help text format list on the adjacent line.
- `config/config.go`: in `NewOutputSettings()` (`config/config.go:87-96`), set `settings.OutputFileFormat = config.GetLCString("output.file-format")`. Use `GetLCString` to match the lowercasing applied to `output.format` via `SetOutputFormat`. In `ShouldCombineAndAppend()` (`config/config.go:71-79`), compare against the effective file format instead of `OutputFormat`.
- `config/config_test.go`: extend `TestConfig_NewOutputSettings` with cases for: file format set, file format absent (asserts `OutputFileFormat` is the empty string — the fallback itself happens inside the library), and case-insensitivity. Add `ShouldCombineAndAppend` cases for a divergent file format. Add one end-to-end test that builds a small `format.OutputArray` from `NewOutputSettings()` with `output.format=json`, `output.file-format=csv`, and a temp file, calls `Write()`, and asserts the file contains CSV — this proves the library integration rather than only the field wiring.

Command help output updates automatically via Cobra. The generated docs under `docs/` are refreshed separately via `awstools gen docs` when the site is updated, not per change.

Dependencies: `github.com/ArjenSchwarz/go-output v1.4.0` (no upgrade needed), Cobra/Viper flag binding patterns already in `cmd/root.go`.

Out of Scope:
- Upgrading to go-output v2 (how fog implements this) — unnecessary for this feature.
- Fixing pre-existing go-output v1 quirks: `html` and `drawio` renderers write directly to the output file inside the library, so `-o html --file x --file-format y` combinations inherit existing odd behaviour; graph formats (`dot`, `mermaid`, `drawio`) terminate with an error on commands that do not configure them. These behave identically today when chosen via `--output`.
- Validating format values for either flag.
- Regenerating the committed Cobra docs under `docs/`.
- Guarding `--append` runs that mix formats in one file across invocations; that is already possible today by varying `--output` between runs and remains the user's responsibility.

## Risks and Assumptions

- Risk: choosing `drawio`, `dot`, or `mermaid` as `--file-format` on a command that does not configure those formats causes the library to exit with an error mid-write. | Mitigation: identical to existing `--output` behaviour for those formats, so no regression; documented as out of scope.
- Risk: `-o html` combined with a different `--file-format` produces surprising results because the v1 HTML renderer writes to the file directly. | Mitigation: pre-existing library behaviour, out of scope; the common direction (readable stdout + machine-readable file) is unaffected.
- Risk: commands whose combine-and-append flow parses the existing file (e.g. `vpc peerings` via `drawio.GetHeaderAndContentsFromFile`) cannot parse a file written in an incompatible format. | Mitigation: same failure exists today when `--output` varies between appending runs; the `ShouldCombineAndAppend` change keeps the HTML exclusion keyed to the right format, the rest stays user responsibility.
- Assumption: go-output v1.4.0's `OutputFileFormat` rendering path works as designed; validated by the end-to-end `Write()` test above rather than only by reading the library source.
- Assumption: `--file-format` without `--file` is a harmless no-op (the library only consults `OutputFileFormat` when `OutputFile` is set).
