---
references:
    - specs/file-output-format/smolspec.md
    - specs/file-output-format/decision_log.md
---
# File Output Format

- [x] 1. The root command exposes --file-format as a global persistent flag (available to every subcommand, like --file), bound to config key output.file-format; the --output help text lists all supported formats

- [x] 2. Output settings carry the configured file format: NewOutputSettings populates OutputFileFormat lowercased, empty when unset; unit tests cover set, unset, and mixed-case values

- [x] 3. Combine-and-append decisions use the effective file format: ShouldCombineAndAppend excludes HTML based on file-format when set, otherwise output format; unit tests cover divergent combinations

- [x] 4. A divergent file format is proven end to end: a Write() test with json stdout and csv file-format produces a CSV file while stdout remains json

- [x] 5. Full suite passes and the feature works from the CLI: go fmt clean, make test green, and demo tables with -o table --file out --file-format json writes a JSON file alongside table stdout
