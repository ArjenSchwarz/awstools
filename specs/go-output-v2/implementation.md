# Implementation Explanation: go-output v2 Migration

Explains the migration of awstools from go-output v1.4.0 to v2.7.0 (branch `worktree-go-output-v2`, 47 commits) at three expertise levels, followed by a completeness assessment.

## Beginner Level

### What Changed

awstools prints everything it finds in AWS — lists of S3 buckets, diagrams of network connections, tables of IAM roles — and it uses a library called go-output to turn that data into JSON, tables, CSV files, and diagrams. This branch replaces version 1 of that library with version 2 across every command in the tool.

Think of go-output as a printing press. Version 1 worked like a typewriter: each command poured its data into one shared buffer and hit "print" as it went. Version 2 works like a print shop: you hand over a finished document, say which format and destination you want, and the shop renders it. Every one of the 24 command files was rewritten to build a document first and hand it to one central "print shop" helper.

### Why It Matters

- Version 1 was a dead end: it mutated shared state, made testing hard, and its draw.io diagram support couldn't read files back for merging data from multiple AWS accounts. Version 2.7.0 adds exactly that read-back feature.
- Users keep the same commands, flags, and (almost entirely) the same output. The few deliberate differences are documented, the biggest being that JSON output now wraps rows in an envelope: what used to be `[...rows...]` is now `{"title": ..., "data": [...rows...], "schema": ...}`. Scripts using `jq '.[]'` change to `jq '.data[]'`.
- Errors are now returned properly instead of the program abruptly exiting, so scripts wrapping awstools get reliable exit behavior.

### Key Concepts

- **Document**: v2's unit of output — an immutable bundle of tables, graphs, or diagram data built once and rendered many ways.
- **Render pipeline**: the central helper in the `config` package that decides which formats go to the screen and which go to a file, applying settings like emoji or table styles in one place.
- **Equivalence oracle tests**: tests that pin down exactly what each output format produces for a fixed sample of data, so any accidental change to output shows up as a test failure.

---

## Intermediate Level

### Changes Overview

- `config/config.go`: new render core. `DocumentSet{Table, Graph, DrawIO}` holds per-format-family document flavors; `RenderDocuments(ctx, docs, opts...)` renders stdout in `output.format` and the file destination in the effective `--file-format`, sequentially, returning (never fataling on) errors. `WithFileOverwrite()` lets combine-style commands suppress append mode after they merged prior file contents themselves. v1's `NewOutputSettings`/`GetSeparator` are deleted.
- `cmd/graph.go` + `cmd/drawio.go`: shared helpers replicating v1 semantics — `graphEdges` (v1's from/to splitting, pinned against v1 source), `awsShape` (lenient shape lookup with stderr warning), `drawIOConnection`/`drawIOBaseHeader` (v1 defaults), `drawIORecords` (comma-joins `[]string` cells so draw.io multi-value references keep working), `drawIORecordString` (read-back accessor). The record/edge helpers are generic over `~map[string]any` so both `[]map[string]any` and `[]output.Record` rows pass directly.
- All 24 command files: `OutputArray`/`OutputSettings` replaced by builder calls (`output.New().Table(title, rows, WithKeys...)`), rendered through `settings.RenderDocument(s)`. Graph-capable commands build their Graph flavor when `NeedsGraphFormat()` and DrawIO flavor when `IsDrawIO()` — additively, on every render path, so the central unsupported-format guard never fires for capable commands. `Run` handlers became `RunE`.
- `names`/`organizations names` dropped go-output entirely (direct JSON write, dead S3 path removed).
- Tests: config render core suite, helper-pinning tests, per-format oracle tests, verbose-dimension and drawio combine-and-append tests, vpcenis rewrite, demotables style-drift guard.
- `go.mod`: only `github.com/ArjenSchwarz/go-output/v2 v2.7.0` remains.

### Implementation Approach

The migration was structured dependency-first: v2 added alongside v1 → central render helper (TDD) → shared cmd helpers (TDD) → command migrations in three behavioral classes (canonical tables, graph+drawio, special cases) → fixture-driven verification → v1 removal. Each command class followed a written pattern in design.md, which kept 24 independently-migrated files convergent.

The core design constraint is that **one document cannot serve divergent format families** — v2 renderers walk all document contents, so a drawio renderer would also serialize table content. Hence `DocumentSet` with per-family flavors and a format→flavor dispatch (`flavorFor`) that reproduces v1's "This command doesn't currently support the X output format" guard for nil flavors.

### Trade-offs

- **Published tag over local replace** (D2): reproducible builds; go-output itself was out of scope.
- **Accept v2 rendering defaults** rather than replicate v1 bytes: CSV multi-value/bool cells (D7), HTML document template (D4), the JSON/YAML `{title,data,schema}` envelope (D11, found and documented during pre-push review — v2.7.0 has no bare-array mode, and a custom unwrapping format would be permanent bespoke renderer code).
- **Fixture oracles over per-command tests** (design/R8.1): coverage is concentrated in the shared pipeline and helpers; individual canonical commands rely on the oracle plus pattern review rather than per-command render tests.
- **Two intentional behavior improvements**: nil graph cells no longer emit v1's `%!s(<nil>)` artifact (D10), and `tgw routes --list` now serves graph/drawio formats instead of erroring misleadingly (mandated by the guard invariant).

---

## Expert Level

### Technical Deep Dive

- **Render core seam**: `renderDocuments(ctx, docs, stdout output.Writer, opts...)` is unexported with an injectable stdout writer; tests capture stdout bytes without `os.Pipe`. The R4.6 invariant (file bytes == stdout bytes when formats match) holds at the renderer layer; v2's `NewStdoutWriter` appends a trailing newline the `FileWriter` doesn't, normalized explicitly in the vpcenis test with a comment.
- **Append semantics**: `WithAppendMode` is applied only when `output.append` is set and the command didn't signal `WithFileOverwrite()`. The signal is a render option, not derived from `ShouldCombineAndAppend()`, because that getter is true for every non-HTML file format and would silently disable the csv/json multi-account append flow. Only `tgw overview` and `vpc peerings` pass it, exactly when they performed the drawio read-back merge (`output.ParseDrawIOFile` keyed by ID, `unique()` dedup of destinations).
- **v1 semantics pinned in helpers**: `graphEdges` replicates the join-then-split quirk (a `[]string` element containing a comma yields extra edges — pinned in tests); `drawIORecords` comma-joins `[]string` cells because v2's drawio renderer stringifies cells with `fmt.Sprint`, which would emit `[a b c]` and break draw.io's multi-value connection refs. The pre-push review caught six commands bypassing `drawIORecords`; all now route through it and the drawio oracle pins a multi-value cell.
- **Determinism (R2.8)**: commands ranging Go maps (`tgwoverview`, `tgwroutes`, `vpcpeerings`, `sso overview-by-account`, `helpers.GetAccountList`) iterate `slices.Sorted(maps.Keys(...))`.
- **Shape drift**: `AWSShape("Network Content Delivery", "Direct Connect Gateway")` is missing from both v1's and v2's shape JSON — v1 already rendered an empty style; `awsShape` preserves that but now warns on stderr. The other 14 command shape pairs are byte-identical between versions.

### Architecture Impact

- The `config` package is now the single choke point for all rendering policy (formats, destinations, transformers, append). Commands own only data-shaping. Adding a format or destination is a one-file change.
- `Run` → `RunE` propagation changed the failure contract: cobra prefixes `Error:` and exits non-zero via `rootCmd.Execute()`; no more mid-render `log.Fatal` leaving partial file writes unreported.
- The JSON/YAML envelope (D11) is the one user-visible API break; `schema.keys` now carries column order, which bare arrays couldn't express.
- go.mod collateral: MVS pulled AWS SDK minor bumps (notably `service/s3` 1.83→1.88.5); `gopkg.in/yaml.v3` became a direct (test-only) dependency.

### Potential Issues

- **Partial-output on guard errors**: `renderDocuments` renders stdout before validating the file destination's flavor, so a capable-stdout/incapable-file combination prints to stdout then errors. Acceptable (errors are returned, exit code non-zero) but observable.
- **First-run combine**: `tgw overview`/`vpc peerings` with `--append` call `ParseDrawIOFile` on the configured file; a missing file surfaces as an error rather than being treated as empty. Mirrors v1's read-back behavior, but worth a real-AWS check in the multi-account workflow.
- **s3 service client jump** (five minor versions) is untested against live AWS; the s3 helpers deserve a production sanity run.
- **HTML append**: appending to a v1-era HTML file is unsupported (marker change); documented in README, but users with standing multi-account HTML reports must regenerate.
- **Namefile reads**: drawio-only runs of three overview commands now also build table rows, multiplying `getName` namefile re-reads (pre-existing inefficiency in `getName`, noted for follow-up caching).

## Completeness Assessment

**Fully implemented**: all 27 spec tasks; every requirement group R1–R9 has supporting implementation and tests (R1 dependency swap verified by grep/build; R3/R4 config wiring unit-tested; R5 drawio adapters + combine tested; R6 names semantics byte-compatible; R7 simplifications done; R9 RunE + guard + empty-output oracles).

**Amended during review**: R2.1 (JSON/YAML envelope accepted as Decision 11) and R8.1 (reworded to match the design's fixture-oracle strategy). Both amendments documented; the spec and tests now agree.

**Known gaps (accepted)**: per-command render tests don't exist for ~17 canonical commands (deliberate per design — oracle + shared-core coverage); the combine-and-append test exercises the shared primitives but re-implements the merge inline rather than invoking each command's glue; real-AWS validation of the s3 helpers and the multi-account combine flow remains a manual follow-up.
