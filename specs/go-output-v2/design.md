# Design: go-output-v2

## Overview

Replace go-output v1.4.0's `OutputArray`/`OutputSettings` model with v2.7.0's immutable `Document` + `Output` across all 24 command files and the config package. One new central helper in `config` builds the right `Output` instances from viper and renders a `Document`; commands construct Documents and convert from `Run` to `RunE`.

Requirement references (R n.m) point at `requirements.md`; decisions (D n) at `decision_log.md`.

## Architecture

### Module change (R1)

- `go.mod`: add `github.com/ArjenSchwarz/go-output/v2 v2.7.0`; remove `github.com/ArjenSchwarz/go-output v1.4.0` and the stale commented `replace ... => ../go-output2` once no file imports v1.
- Import convention: `output "github.com/ArjenSchwarz/go-output/v2"` (the module's package name is `output`; alias it explicitly for clarity), plus `"github.com/ArjenSchwarz/go-output/v2/icons"` where AWS shapes are needed.
- go-output's automated `migrate` tool is **not** used: it targets the v1 `/format` import layout, and awstools' dominant patterns (settings-hub config, graph/drawio branches, combine flows) are exactly the ~20% it doesn't handle. Manual migration per the class table below.
- Committed cobra docs under `docs/` are regenerated separately via `awstools gen docs` when the site is updated (requirements Non-Goals); no step here. `make test` (lint + tests) must pass with the new `cmd/graph.go`/`cmd/drawio.go` files — no Makefile changes expected.

### Central render helper (replaces `config.NewOutputSettings`)

Lives in `config/config.go` as methods on `*Config` (D8). Commands access it through the existing `settings` global in `cmd/root.go:15`.

**One Document cannot serve divergent format families.** Verified in v2 source (`graph_renderers.go:611-616`): every renderer walks *all* document contents — the drawio renderer renders a `TableContent` *in addition to* a `DrawIOContent` (default header, duplicate nodes), and tabular renderers would serialize graph/drawio content into JSON/tables. A Document handed to a format-F Output must therefore contain only F-appropriate content. Commands supply per-family flavors:

```go
// DocumentSet holds per-format-family document flavors. Table is required;
// Graph/DrawIO are nil unless the command supports those formats.
type DocumentSet struct {
    Table  *output.Document // json/yaml/csv/table/markdown/html
    Graph  *output.Document // dot/mermaid
    DrawIO *output.Document // drawio
}

// RenderDocuments picks the flavor for each destination's format and renders:
// stdout in output.format, then output.file in the effective file format when set.
// Returns the first error; never exits.
func (config *Config) RenderDocuments(ctx context.Context, docs DocumentSet) error

// RenderDocument is sugar for the ~15 table-only commands:
// RenderDocuments(ctx, DocumentSet{Table: doc}).
func (config *Config) RenderDocument(ctx context.Context, doc *output.Document) error

// internal pieces
func (config *Config) stdoutOutput() *output.Output            // format(output.format) + StdoutWriter + transformers
func (config *Config) fileOutput() (*output.Output, error)     // format(effective file format) + FileWriter [+ append]
func formatFor(name string, config *Config) output.Format       // name -> Format, fallback JSON (R3.1, R3.6)
```

Behavioral contract of `RenderDocuments`:

1. **Two-Output dispatch** (R4.1–4.3): v2 renders every format to every writer (cross-product), so stdout and file each get their own `Output`, each over the flavor matching its format. Rendered sequentially: stdout first, then file. When the effective file format equals the stdout format both destinations get the same flavor and the render is deterministic, satisfying R4.6 without extra machinery.
2. **Format table** (`formatFor`):

   | config value | v2 Format |
   |---|---|
   | `json` (default), unknown | `output.JSON()` |
   | `yaml` | `output.YAML()` |
   | `csv` | `output.CSV()` |
   | `table` | `output.TableWithStyleAndMaxColumnWidth(style, width)` from `output.table.style` / `output.table.max-column-width` (R3.3) |
   | `markdown` | `output.Markdown()` |
   | `html` | `output.HTMLWithTemplate(output.DefaultHTMLTemplate)` (D4) |
   | `dot` | `output.DOT()` |
   | `mermaid` | `output.Mermaid()` |
   | `drawio` | `output.DrawIO()` |

3. **Graph-format guard** (R9.2): when a destination's format is `dot`/`mermaid` but `docs.Graph` is nil, or `drawio` but `docs.DrawIO` is nil, error with v1's text: `This command doesn't currently support the <format> output format`. This is a nil-flavor check — no document walking — and replicates v1's `FromToColumns == nil` / `DrawIOHeader.IsSet()` guards (v1 `output.go:165-177`) exactly: capable commands populate the flavor under the same predicate the guard checks (so it never fires for them); incapable commands leave it nil. **Invariant: every render path of a capable command must populate its graph/drawio flavor when the predicate holds** — e.g. `tgw routes`' `simplelistOnly` path either builds the graph flavor too or the guard fires with a misleading message; the implementation must cover both paths.
4. **File writer**: `output.NewFileWriterWithOptions(filepath.Dir(f), filepath.Base(f), opts...)`; the pattern contains no `{format}`/`{ext}` placeholders so the exact filename is preserved. `output.WithAppendMode()` is added when `output.append` is true, **except** when the command explicitly signals it already merged the prior file content into the Document. The signal is a render option, *not* derived from `ShouldCombineAndAppend()` — that getter returns true for every non-html file format when append is set, so using it globally would disable append for the csv/json multi-account flow (silent overwrite):

   ```go
   func (config *Config) RenderDocuments(ctx context.Context, docs DocumentSet, opts ...RenderOption) error
   func WithFileOverwrite() RenderOption // combine commands pass this when they performed the read-back merge
   ```

   Only `tgw overview` and `vpc peerings` pass `WithFileOverwrite()`, and only when `settings.ShouldCombineAndAppend()` was true for their run (i.e. exactly when they merged). Known behavior: v2's HTML append inserts at a `<!-- go-output-append -->` marker, incompatible with HTML files produced by v1 (`<div id='end'></div>`); appending to a pre-migration HTML file requires regenerating it (documented v2 breaking change, acceptable under D3/D4).
5. **Transformers**: `output.WithTransformer(&output.EmojiTransformer{})` on both Outputs when `output.use-emoji` (R3.2). Emoji in v2 is presentation-only (table/markdown/html/csv; never json/yaml). v1's bool→emoji conversion disappears with v2's `true`/`false` bool rendering — accepted under D7; `tgw overview`'s manual ✅/❌ prefixes keep working because that command writes emoji into the data itself (it switches from reading `Settings.UseEmoji` to `settings.GetBool("output.use-emoji")`).
6. **Sorting** (R3.4) is *not* an Output concern in v2: commands attach `output.WithTransformations(output.NewSortOp(output.SortKey{Column: k, Direction: output.Ascending}))` per table. A one-line helper in `config` keeps call sites short:

```go
// SortOption returns a TableOption sorting by column, ascending — v1 SortKey equivalent.
func SortOption(column string) output.TableOption
```

v2 sorts with typed comparison (stable, nils first) where v1 compared stringified values; awstools' sort columns are all strings, so behavior matches — the per-command equivalence tests pin this.

7. **Test seam**: the public entry points wrap an unexported core `renderDocuments(ctx, docs, stdout output.Writer, opts...)`; production passes `output.NewStdoutWriter()`, config-package tests inject a buffer writer to capture "stdout" bytes (needed for the R4.6 file==stdout assertion and the stdout-format tests without `os.Pipe` gymnastics).

`Config` keeps its viper getters and `ShouldCombineAndAppend`/`IsDrawIO`/`IsVerbose` (unchanged semantics, R4.5). `GetSeparator()` (dead, zero callers) and `NewOutputSettings()` are deleted.

### Command conversion: `Run` → `RunE` (R9.1, D9)

All command definitions switch to `RunE`; body `panic(err)` / `log.Fatal(err)` become `return err`. `rootCmd.Execute()` already exits non-zero on error (`cmd/root.go:35`). Helpers in `helpers/` are out of scope. The detail functions invoked by commands (e.g. `ssoOverviewByAccount(cmd, args)`) change signature to return `error`.

### Command migration classes

Every v1-using file, its class, and what changes:

| File | Class | Notes |
|---|---|---|
| `cmd/appmeshdanglingnodes.go` | canonical | single table |
| `cmd/appmeshmeshroute.go` | canonical | verbose-conditional keys (R2.6) |
| `cmd/cfnresources.go` | canonical | verbose adds columns |
| `cmd/s3list.go` | canonical | |
| `cmd/ssodangling.go` | canonical | |
| `cmd/ssolistpermissionsets.go` | canonical | sort key |
| `cmd/tgwdangling.go` | canonical | |
| `cmd/sso.go` | canonical | table in `displayEnhancedProfileResults` |
| `cmd/vpcroutes.go` | canonical | raw `[]string` cells → v2 default rendering (D7); drawio remnants stay commented |
| `cmd/vpcipfinder.go` | canonical | missed by the original table; added during phase review (task 23) |
| `cmd/appmeshshowmesh.go` | graph+drawio | |
| `cmd/iamrolelist.go` | graph+drawio | heterogeneous rows (roles + policies in one table) |
| `cmd/iamuserlist.go` | graph+drawio | heterogeneous rows; two connections |
| `cmd/organizationsstructure.go` | graph+drawio | helper appends rows via `*builder` instead of `*OutputArray` |
| `cmd/ssooverviewaccount.go` | graph+drawio | v1 reassigns `Keys` mid-build → two explicit key sets |
| `cmd/ssooverviewpermissionset.go` | graph+drawio | same |
| `cmd/tgwroutes.go` | graph+drawio | two render paths (full / `simplelistOnly`) |
| `cmd/vpcpeerings.go` | graph+drawio+combine | non-default connection style string (R5.2) |
| `cmd/tgwoverview.go` | drawio+combine | emoji-in-data |
| `cmd/vpcoverview.go` | multi-table | 3 arrays + per-loop `Write()` → one Document, one render (R7.3) |
| `cmd/vpcenis.go` | multi-table | delete `AddToBuffer`/`combined` choreography (R7.2); keep `enisGraphFormatError` guard, which fires pre-build so the central nil-flavor guard is unreachable for this command — its message stays as-is (R9.2) |
| `cmd/demotables.go` | special | style-map iteration → hardcoded style list (R7.4) |
| `cmd/names.go` | raw JSON | drop `PrintByteSlice`/S3 (R6) |
| `cmd/organizationsnames.go` | raw JSON | same |
| `cmd/organizationsnames.go`/`names.go` shared | | merge-on-append logic unchanged |
| `config/config.go` | core | helper above |
| `config/config_test.go` | tests | rewritten (see Testing) |

**Canonical pattern** (before → after):

```go
// v1
output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
output.Settings.Title = "Title"
for ... { output.AddHolder(format.OutputHolder{Contents: content}) }
output.Write()

// v2
rows := []map[string]any{}            // was []OutputHolder
for ... { rows = append(rows, content) }
doc := output.New().
    Table("Title", rows, output.WithKeys(keys...)).
    Build()
return settings.RenderDocument(cmd.Context(), doc)
```

Row maps stay `map[string]any` built exactly as today (naming resolution untouched). v1's `Title` becomes the table title (R3.5). `cmd.Context()` supplies the context.

**Multi-table** (`vpcoverview`, `vpcenis`): accumulate `.Table(...)` calls on one builder (one per subnet/VPC group, each with its own `WithKeys` — R2.7), single `RenderDocument`. `SeparateTables`, `AddToBuffer`, per-loop `Write()` all deleted. Known shape change: v1's per-loop `Write()` emitted *concatenated* JSON documents (one per call, invalid as a whole); v2 emits one JSON document containing all tables — an accepted consequence of R7.3/D5, and strictly more parseable.

**Heterogeneous rows** (`iamrolelist`, `iamuserlist`): v1 put rows with different key subsets in one array; keys were the union. Preserved as-is: one table, union `WithKeys`, absent cells render empty — identical to v1.

**Keys-reassigned-mid-build** (`ssooverview*`): v1 mutated `output.Keys` before `Write()` in the drawio branch. v2 makes the two shapes explicit as separate DocumentSet flavors. **Flavor construction is additive, never exclusive**: the Table flavor is always built; the DrawIO flavor is built *in addition* whenever `IsDrawIO()` holds (likewise Graph under `NeedsGraphFormat()`). With `--output table --file-format drawio` both destinations must be served — an exclusive if/else would starve one of them.

### Graph (dot/mermaid) construction

v2's table→graph auto-detect matches only `from/source/start` × `to/target/end/dest` column names — none of awstools' pairs (`Name`→`Endpoints`, `ID`→`PeeringIDs`, …), and `NewGraphContentFromTable` stringifies a `[]string` "to" cell into one bogus node instead of one edge per element (v1 `splitFromToValues` split on `,`). Both facts force explicit edge construction:

```go
// cmd/graph.go (new, small)
// graphEdges replicates v1 splitFromToValues: one edge per comma-separated
// element of the "to" cell; []string cells are joined first; empty targets skipped.
// Implementation note: pin v1's exact []string join semantics with a unit test
// (v1 stringified the cell, then split on ",").
func graphEdges(rows []map[string]any, fromCol, toCol string) []output.Edge
```

Graph content lives in its own DocumentSet flavor (never mixed into the table Document — renderers walk all contents, so mixing would leak graph data into JSON and duplicate nodes in graph output). Each of the 8 v1 `AddFromToColumns` call sites becomes:

```go
docs := config.DocumentSet{Table: tableDoc}
if settings.NeedsGraphFormat() {   // new Config helper: stdout OR effective file format is dot/mermaid
    docs.Graph = output.New().Graph(title, graphEdges(rows, fromCol, toCol)).Build()
}
```

**Format-need checks consult both destinations**: v1's `IsDrawIO()` inspected only the stdout format, so `--output json --file-format drawio` never built drawio content and v1 `log.Fatal`ed (a quirk the `file-output-format` spec documented as inherited). v2 extends the checks — `NeedsGraphFormat()` and `IsDrawIO()` return true when *either* the stdout format or the effective file format matches — so those combinations now work on capable commands. This is a deliberate improvement consistent with R4.2; incapable commands still error via the guard, preserving R9.2.

### Draw.io construction

New file `cmd/drawio.go` with three small adapters:

```go
// awsShape wraps icons.GetAWSShape, returning "" and logging a warning on unknown shapes (R5.3).
func awsShape(group, title string) string

// drawIOConnection returns v1 NewConnection defaults:
// {From: "Parent", To: "Name", Invert: true, Style: output.DrawIODefaultConnectionStyle}
func drawIOConnection() output.DrawIOConnection

// drawIOBaseHeader returns DefaultDrawIOHeader with Height/Width "78" —
// the shared shape every command's create*DrawIOHeader builds on.
func drawIOBaseHeader(label, style, ignore string) output.DrawIOHeader
```

Mapping table (mechanical, per `create*DrawIOHeader` function):

| v1 | v2 |
|---|---|
| `drawio.NewHeader(l, s, i)` | `drawIOBaseHeader(l, s, i)` |
| `drawio.DefaultHeader()` | `output.DefaultDrawIOHeader()` |
| `h.SetHeightAndWidth("78","78")` | `.Height = "78"; .Width = "78"` |
| `h.SetLayout(drawio.LayoutX)` | `.Layout = output.DrawIOLayoutX` |
| `h.SetIdentity(c)` | `.Identity = c` |
| `h.AddConnection(c)` | `.Connections = append(...)` |
| `drawio.NewConnection()` | `drawIOConnection()` |
| `drawio.BidirectionalConnectionStyle` | `output.DrawIOBidirectionalConnectionStyle` |
| `drawio.AWSShape(g, t)` (~40 sites) | `awsShape(g, t)` |

Commands add drawio content via `builder.DrawIO(title, records, header)` where `records` is `[]output.Record`. `Record` is a *defined type* (`type Record map[string]any`), not an alias — `[]map[string]any` does not convert implicitly. Drawio-capable commands declare their row slices as `[]output.Record` directly (a `map[string]any` literal assigns to `Record` fine; only the slice type needs declaring). Since the drawio row set sometimes differs from the table rows (extra `Image`/`DrawioID` columns), commands keep their existing `if settings.IsDrawIO()` branches to decide which columns to populate, and always populate them into the same row maps as today (the drawio renderer ignores unknown columns via the header's `Ignore`).

**Combine-and-append port** (`tgwoverview.go:127`, `vpcpeerings.go:48`; R5.4):

```go
// v1
headers, previousResults := drawio.GetHeaderAndContentsFromFile(file)
id := row[headers["ID"]]                       // positional

// v2
parsed, err := output.ParseDrawIOFile(file)     // error now surfaced (RunE)
id := rec["ID"]                                 // parsed.Records: keyed
```

The ID-keyed merge/dedup logic around it is awstools-owned and unchanged. The Document then contains merged records and the file is written *fresh* (append mode suppressed for the combine case, see helper contract #4). Column order stability across round-trips: pass `output.WithDrawIOColumns(cols...)` when re-emitting is unnecessary here because the commands rebuild records with their own fixed key sets — order comes from the header/keys the command sets, same as v1.

### Raw JSON name-file commands (R6)

`names.go` / `organizationsnames.go`: keep `json.Marshal` + existing merge-on-append; replace `format.PrintByteSlice(b, file, s3)` with a direct write that mirrors v1's exact semantics: write to `output.file` when set, *otherwise* print to stdout (v1 wrote to one destination, not both). No go-output involvement (matches v1 reality), S3 argument dropped (D6, R6.3).

### demotables (R7.4)

`format.TableStyles` map is unexported in v2. Hardcode the 16 v1 style names (`Default`, `Bold`, `ColoredBright`, …) as a slice; loop renders one single-table Document per style through an `Output` built with `output.TableWithStyle(name)`. Uses its own inline Output construction rather than `RenderDocument` (it deliberately varies the style per iteration); `--file` support is intentionally dropped for this demo command. A test asserts each hardcoded name is accepted by `output.TableWithStyle` so the list cannot silently drift from v2.

## Error Handling

- `RenderDocument` returns errors (render failures, unwritable file, graph-format guard); commands `return err` under `RunE`; cobra prints to stderr and `Execute()` exits non-zero (R9.1).
- `ParseDrawIOFile` errors (missing/corrupt prior file) propagate the same way — v1 silently produced `(nil, nil)` on some failures; surfacing the error is an accepted improvement consistent with R9.1.
- `awsShape` swallows unknown-shape errors (warning to stderr, empty style) to match v1's empty-string behavior (R5.3).
- Empty documents: v2 renders zero-row tables as valid empty output per format; no special casing (R9.3).

## Testing Strategy

Test files live beside the code (`cmd/*_test.go`, `config/config_test.go`).

1. **config helper tests** (rewrite of `config_test.go`): viper-driven table tests for `formatFor` (every name + unknown fallback, R3.1/R3.6), transformer/style/width wiring (R3.2/R3.3), and an end-to-end `RenderDocument` with `output.format=json`, `output.file-format=csv`, temp file: assert stdout JSON parses with expected keys, file parses as CSV (R8.3 / AC 4.2). Add the formats-match case asserting file bytes == captured stdout bytes (R4.6).
2. **Equivalence oracles** (R8.1), one representative per family, driven by fixed in-memory rows (no AWS calls):
   - json/yaml: deep-equal fields via unmarshal; key order asserted from the byte stream with `json.Decoder` tokens (unmarshalling into maps loses order).
   - csv: parse, assert header order + scalar cells (multi-value cells asserted against v2's documented rendering, D7).
   - table/markdown: assert column headers and row values appear in order (contains-based, styling-agnostic).
   - dot/mermaid: assert exact node set + directed edge set parsed from output (R2.4); `graphEdges` unit-tested against v1 semantics: `[]string` cell → N edges, comma-string → split, empty → skipped.
   - drawio: render a Document with known header/records, assert via `output.ParseDrawIOCSV` round-trip that nodes/shapes/connections survive (R5.1/5.2).
   - html: assert data presence within the templated document (R2.5).
3. **Verbose dimension** (R8.2): one command's row/key builder (e.g. `cfnresources`) exercised verbose on/off, asserting the column delta.
4. **Draw.io combine** (R8.4): write a drawio CSV via v2, run the merge logic over it plus new records, assert combined ID set and column order (guards `vpcpeerings`/`tgwoverview` port).
5. **Guard tests** (R9.2): `RenderDocument` with `output.format=dot` and a table-only Document errors with the v1-compatible message; `vpc enis` keeps its own guard test.
6. **Existing tests**: `cmd/vpcenis_test.go` (T-1294 coverage) is rewritten against the new single-Document flow, preserving its intent: file output contains all subnet tables.

Property-based testing: not added here — the round-trip properties worth PBT (draw.io CSV parse/render) are already property-tested inside go-output v2.7.0 itself (`pgregory.net/rapid`); duplicating them in awstools would test the library, not the migration.
