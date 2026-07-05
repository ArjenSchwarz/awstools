# Requirements: go-output-v2

## Introduction

awstools renders all command output through go-output v1.4.0, using its `OutputArray`/`OutputSettings` buffer model and the `drawio` subpackage. go-output v2 replaces that model with an immutable `Document` plus a configurable `Output` (formats + writers + transformers), and v2.7.0 adds the draw.io CSV read-back that awstools needs for its append/combine flow. This feature migrates every awstools command to go-output v2.7.0 while keeping command behavior and per-format output functionally equivalent, and removes the v1 dependency once complete.

## Non-Goals

- Changing any CLI flag, config key, or output-format name.
- Adding output features beyond v1 parity (charts, collapsible content, S3 append, progress bars).
- Wiring real S3 output — the existing S3 path is dead (no flag binds it); it is removed, not implemented.
- Changing awstools-local naming resolution (`getName` / `--namefile`).
- Byte-for-byte output equality; equivalence is at the data level (see Requirement 2).
- Restoring v1-style CSV/table cell formatting for multi-value and boolean columns — v2's default rendering is accepted here and revisited under a separate follow-up task (decision log Decision 7).
- Regenerating the committed Cobra docs under `docs/`.

## Requirements

### 1. Dependency migration to go-output v2.7.0

**User Story:** As a maintainer, I want awstools to depend on go-output v2.7.0 instead of v1.4.0, so that the project is on the supported, actively developed major version.

**Acceptance Criteria:**

1. <a name="1.1"></a>The system SHALL depend on `github.com/ArjenSchwarz/go-output/v2` at the published `v2.7.0` tag, resolved through the module proxy without a local `replace` directive.  
2. <a name="1.2"></a>WHEN the migration is complete, the system SHALL NOT reference `github.com/ArjenSchwarz/go-output` v1 or its `drawio` subpackage anywhere in the build, and `go.mod`/`go.sum` SHALL NOT list the v1 module.  
3. <a name="1.3"></a>The system SHALL build and pass `go fmt`, `go vet`, `go test ./...`, and `make test` after migration.  
4. <a name="1.4"></a>The stale commented `replace ... => ../go-output2` directive SHALL be removed.  

### 2. Per-format output equivalence

**User Story:** As an awstools user, I want every command's output to carry the same data after the migration, so that my existing usage and downstream tooling keep working.

**Acceptance Criteria:**

1. <a name="2.1"></a>For the `json` and `yaml` formats, the system SHALL produce row data with the same fields, the same native value types (arrays stay arrays, booleans stay booleans), and the same key/column order as v1; JSON MAY be pretty-printed where v1 emitted compact JSON. The rows move from v1's bare top-level array into v2's document envelope (`{"title": ..., "data": [rows], "schema": {"keys": [...]}}`) — an accepted breaking change to the wrapper, not the row data (Decision 11).  
2. <a name="2.2"></a>For the `csv` format, the system SHALL produce the same columns in the same order, with the same scalar cell values, as v1; multi-value (list) and boolean cells adopt go-output v2's default rendering (e.g. `[a b c]`, `true`/`false`) — a known divergence from v1's newline-joined lists and `Yes`/`No` booleans, accepted and tracked for follow-up review (decision log Decision 7).  
3. <a name="2.3"></a>For the `table` and `markdown` formats, the system SHALL produce the same columns, column order, and row data as v1; cell styling and multi-value/boolean cell formatting MAY differ where v2 renders differently.  
4. <a name="2.4"></a>For the `dot`, `mermaid`, and `drawio` formats, the system SHALL produce the same set of nodes, the same set of directed edges, and (for drawio) the same AWS shape assignments as v1.  
5. <a name="2.5"></a>For the `html` format, the system SHALL emit the same tabular data as v1 wrapped in go-output v2's default responsive HTML document template; the surrounding document structure MAY differ from v1's bare fragment.  
6. <a name="2.6"></a>Each equivalence criterion above SHALL hold with `--verbose` both enabled and disabled, since verbose mode changes which columns and rows appear in roughly nine commands.  
7. <a name="2.7"></a>WHERE a command renders multiple tables, the system SHALL render each table with its own column schema (no shared or unioned schema across tables), preserving the per-table headers v1 produced.  
8. <a name="2.8"></a>WHERE a table is not explicitly sorted, the system SHALL preserve v1's row order (insertion order of the underlying data); commands that build rows by ranging a Go map SHALL impose a deterministic order so output does not vary between runs.  

### 3. Output configuration from existing flags and config

**User Story:** As an awstools user, I want the existing output flags and config keys to keep controlling output the same way, so that nothing in my command lines or `.awstools.yaml` changes.

**Acceptance Criteria:**

1. <a name="3.1"></a>The system SHALL honor `--output` / `output.format` to select the stdout format, defaulting to `json`, with unknown values handled exactly as today (no validation error).  
2. <a name="3.2"></a>The system SHALL honor `--emoji` / `output.use-emoji` so that emoji substitution appears in rendered output when enabled and not when disabled.  
3. <a name="3.3"></a>The system SHALL honor `output.table.style` and `output.table.max-column-width`, applying the same named table style and maximum column width as v1 for the same config values.  
4. <a name="3.4"></a>WHERE a command sets a sort key, the system SHALL order that table's rows by that key as v1 did.  
5. <a name="3.5"></a>WHERE a command sets a title for its output, the system SHALL render an equivalent heading for that content.  
6. <a name="3.6"></a>WHEN `--file-format` is set to a value the library does not recognize, the system SHALL apply the same fallback as an unrecognized `--output` value (AC [3.1](#3.1)), with no validation error, so the two flags behave identically.  

### 4. File output and divergent file format

**User Story:** As an awstools user, I want `--file` and `--file-format` to keep working, so that I can write a file in a different format than stdout without running the command twice.

**Acceptance Criteria:**

1. <a name="4.1"></a>WHEN `--file` / `output.file` is set, the system SHALL write the command output to that file in addition to stdout.  
2. <a name="4.2"></a>WHEN `--file-format` / `output.file-format` is set, the system SHALL render the file in that format while stdout continues to use `--output`.  
3. <a name="4.3"></a>WHEN `--file-format` is not set, the file SHALL use the same format as stdout.  
4. <a name="4.4"></a>WHEN `--append` / `output.append` is set, the system SHALL append to the existing file rather than replacing it, matching v1's append behavior for the chosen file format.  
5. <a name="4.5"></a>The combine-and-append flow's HTML exclusion (`Config.ShouldCombineAndAppend`) SHALL remain keyed to the effective file format (`--file-format` when set, otherwise `--output`).  
6. <a name="4.6"></a>WHEN the file format equals the stdout format, the file content SHALL be identical to the stdout content for the same run (guarding against the v1 class of bug where `--file` came out empty while stdout was populated).  

### 5. Draw.io diagram parity

**User Story:** As an awstools user, I want the draw.io-producing commands to keep generating the same diagrams, so that my architecture imports into draw.io unchanged.

The draw.io-emitting commands are the nine that assign a draw.io header: `appmesh show-mesh`, `iam rolelist`, `iam userlist`, `organizations structure`, `sso overview-by-account`, `sso overview-by-permissionset`, `tgw overview`, `tgw routes`, and `vpc peerings`.

**Acceptance Criteria:**

1. <a name="5.1"></a>For each of the nine draw.io commands, the system SHALL emit draw.io CSV with the same node rows, AWS shape styles, and header directives (label, style, layout, identity, height, width) as v1.  
2. <a name="5.2"></a>The system SHALL emit the same node-to-node connections as v1, including connection direction (invert), label, and style, including the one command (`vpc peerings`) that uses a non-default connection style string.  
3. <a name="5.3"></a>WHEN an AWS shape lookup is requested for a group/title that exists, the system SHALL produce the same shape style string v1 produced; WHEN the group/title does not exist, the system SHALL behave equivalently to v1 (no crash; missing shape handled).  
4. <a name="5.4"></a>WHEN combine-and-append reads back an existing draw.io CSV file (`tgw overview`, `vpc peerings`), the system SHALL recover the prior rows by column name and feed them into the command's existing merge logic (keyed by resource ID, de-duplicated as today), so the appended diagram combines prior and new data exactly as v1 did; node de-duplication SHALL remain controlled by the command, not changed by the library.  

### 6. Raw JSON name-file output

**User Story:** As an awstools user, I want the `names` and `organizations names` commands to keep writing the same name-map JSON, so that my naming files are unchanged.

**Acceptance Criteria:**

1. <a name="6.1"></a>The `names` and `organizations names` commands SHALL write the same JSON name-map content to stdout and to `--file` as v1 for the same inputs.  
2. <a name="6.2"></a>WHEN `--append` applies, these commands SHALL merge with the existing name file as they do today.  
3. <a name="6.3"></a>The previously dead S3 output path in these commands SHALL be removed without changing file or stdout behavior.  

### 7. Behavior-preserving simplifications

**User Story:** As a maintainer, I want the v1 buffer workarounds removed where v2's model makes them unnecessary, so that the output code is simpler without changing what users see.

**Acceptance Criteria:**

1. <a name="7.1"></a>The system SHALL render each command's stdout and file output from a single constructed document, producing the same content for both destinations in one pass.  
2. <a name="7.2"></a>The `vpcenis` command SHALL produce the same per-subnet tables on stdout and in the file as v1 without the v1 buffer-reset workaround.  
3. <a name="7.3"></a>The `vpc overview` command SHALL produce the same set of tables as v1 (subnet overview, IP details, summary) without relying on repeated stdout-append render calls.  
4. <a name="7.4"></a>The `demo tables` command SHALL render one table per supported named table style, covering the same styles v1 iterated.  

### 8. Verification

**User Story:** As a maintainer, I want the migration's output parity verified, so that regressions are caught before release.

**Acceptance Criteria:**

1. <a name="8.1"></a>The system SHALL include tests that assert data-level output equivalence (per Requirement 2) for every output format, rendered through the production go-output pipeline; per the design's testing strategy these are fixture-driven per-format oracle tests (fixed in-memory rows rendered through `config.DocumentSet`) rather than per-command AWS data, and the JSON path is exercised through the renderer (not `names`/`organizations names`, which marshal JSON directly).  
2. <a name="8.2"></a>The equivalence tests SHALL include at least one command exercised with `--verbose` enabled, covering the verbose column/row changes (AC [2.6](#2.6)).  
3. <a name="8.3"></a>The system SHALL include a test proving the divergent `--file-format` path writes the file in the requested format while stdout uses `--output` (replacing the equivalent v1 test).  
4. <a name="8.4"></a>The system SHALL include a test covering the draw.io combine-and-append read-back-and-merge path (AC [5.4](#5.4)).  

### 9. Error and edge-case behavior

**User Story:** As an awstools user, I want failures and empty results handled the same way after the migration, so that scripts wrapping awstools keep behaving predictably.

**Acceptance Criteria:**

1. <a name="9.1"></a>WHEN rendering or writing output fails (for example an unwritable file path), the system SHALL report the error and exit non-zero rather than exiting successfully with no output.  
2. <a name="9.2"></a>WHERE a command rejects a graph-only format today (e.g. `vpc enis` rejecting `dot`/`drawio` via its existing guard), the system SHALL preserve that accept/reject behavior per command rather than changing which format/command combinations error.  
3. <a name="9.3"></a>WHEN a command produces zero result rows, the system SHALL emit valid output for the selected format (e.g. an empty JSON array, a header-only or empty table, a valid empty HTML document) without erroring, consistent with v1's empty-result behavior.  
