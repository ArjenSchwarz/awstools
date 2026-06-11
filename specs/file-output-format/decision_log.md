# Decision Log: File Output Format

## Decision 1: Use go-output v1.4.0's built-in OutputFileFormat

**Date**: 2026-06-11
**Status**: accepted

### Context

T-1535 asks for a separate output format for the file written via `--file`, like fog's `--file-format` flag. fog implements this on go-output v2 with multiple writers. awstools is pinned to go-output v1.4.0.

### Decision

Wire up `OutputSettings.OutputFileFormat`, which already exists in go-output v1.4.0 and is already honoured by `OutputArray.Write()`. Do not migrate to go-output v2.

### Rationale

The v1 library already renders the file separately using `OutputFileFormat`, falling back to the stdout format when unset. awstools only needs to expose a flag and set one field. A v2 migration would touch every command in `cmd/` for no functional gain on this ticket.

### Alternatives Considered

- **Migrate to go-output v2 (fog's approach)**: Cleaner multi-writer model - Rejected because it rewrites the output path of every command to deliver a one-field feature.
- **Render the file in awstools itself**: Format the contents twice in command code - Rejected because the library already does exactly this; duplicating it adds maintenance burden.

### Consequences

**Positive:**
- ~10 lines of production code across two files.
- Behaviour identical to the library's tested fallback path when the flag is unset.

**Negative:**
- Inherits v1 quirks: `html`/`drawio` renderers write directly to the file, and graph formats exit with an error on commands that do not configure them.

---

## Decision 2: Key ShouldCombineAndAppend's HTML exclusion to the effective file format

**Date**: 2026-06-11
**Status**: accepted

### Context

`Config.ShouldCombineAndAppend()` (config/config.go:71) gates the multi-account combine-and-append flow and excludes HTML, because HTML appending is handled inside the library's renderer. The exclusion currently checks the stdout format, which was also the file format until now. With `--file-format`, the two can diverge and the check is about the file.

### Decision

Evaluate the HTML exclusion against the effective file format: `output.file-format` when set, otherwise `output.format`.

### Rationale

The function exists solely to decide how the output file is appended to; consulting the stdout format would give wrong answers exactly when the new flag is used (e.g. `-o csv --file-format html` would attempt to parse an HTML file as CSV rows).

### Alternatives Considered

- **Leave it keyed to the stdout format and document the combination as undefined**: Zero code change - Rejected because the fix is a few lines and avoids a foreseeable parse failure.
- **Block `--append` together with `--file-format` entirely**: Simple guard - Rejected because same-format appends (the multi-account collection use case) remain valid and useful.

### Consequences

**Positive:**
- Combine-and-append behaves correctly whichever flag carries the HTML format.

**Negative:**
- Appending runs that mix incompatible formats in one file remain possible and remain the user's responsibility (unchanged from today, where varying `--output` between runs does the same).

---

## Decision 3: No validation of --file-format values

**Date**: 2026-06-11
**Status**: accepted

### Context

`--output` performs no validation; unknown values fall back to JSON inside the library's `Write()`. The new flag could validate against the known format list instead.

### Decision

`--file-format` inherits the same no-validation behaviour as `--output`.

### Rationale

Two flags accepting the same value set should fail (or not) the same way. Adding validation to only the new flag creates an inconsistency, and adding it to both expands scope beyond this ticket.

### Alternatives Considered

- **Validate against the supported format list**: Earlier, clearer errors - Rejected for this ticket to keep behaviour consistent with `--output`; could be done for both flags as a follow-up.

### Consequences

**Positive:**
- Consistent flag behaviour; no new failure mode.

**Negative:**
- A typo in `--file-format` silently produces a JSON file.

---
