# Decision Log: go-output-v2

## Decision 1: Full spec workflow (not smolspec)

**Date**: 2026-06-22
**Status**: accepted

### Context

Migrating awstools from go-output v1.4.0 to v2.7.0 touches the output path of every command. Scope assessment found 26 affected files (all of `cmd/`, plus `config/config.go` and `config/config_test.go`), a breaking API swap, cross-cutting impact, and backward-compatibility risk across nine output formats.

### Decision

Use the full spec workflow (requirements → design → tasks), not smolspec.

### Rationale

Every full-spec trigger applies: >80 LOC, >3 files, multiple subsystems, breaking API change, backward-compatibility implications. This is far past the smolspec threshold.

### Alternatives Considered

- **Smolspec**: Lightweight single-doc spec - Rejected; reserved for <80 LOC / 1-3 file changes with no cross-cutting concerns.

### Consequences

**Positive:**
- Design decisions (central output helper, format/writer dispatch) are captured before implementation.

**Negative:**
- More upfront documentation before coding.

---

## Decision 2: Migrate against the published v2.7.0 tag, not a local replace

**Date**: 2026-06-22
**Status**: accepted

### Context

go-output v2 is developed locally at `../go-output/v2`; v2.7.0 is also a published tag. awstools could build against either via a `replace` directive or `go get`.

### Decision

Depend on `github.com/ArjenSchwarz/go-output/v2@v2.7.0` resolved through the module proxy, with no `replace` directive. Remove the stale commented `replace ... => ../go-output2`.

### Rationale

Published tag gives reproducible builds and a clean `go.mod`. go-output is not being changed as part of this work, so co-development isolation is unnecessary.

### Alternatives Considered

- **Local replace `=> ../go-output/v2`**: Enables co-developing both repos - Rejected; not needed here and pollutes `go.mod` for other contributors/CI.

### Consequences

**Positive:**
- Reproducible builds; standard dependency resolution.

**Negative:**
- Any go-output fix needed mid-migration must be tagged/released first.

---

## Decision 3: Functional equivalence, not byte-for-byte output

**Date**: 2026-06-22
**Status**: accepted

### Context

v2 renderers differ from v1 in subtle ways (HTML document wrapping, multi-value cell formatting, table styling). Requiring byte-identical output would forbid v2 improvements and is unachievable for some formats.

### Decision

The acceptance bar is data-level (functional) equivalence: same fields, values, and column/key order per format. Minor formatting differences are allowed where v2 renders differently; HTML may change structurally (see Decision 4). Verification asserts data equivalence, not byte equality.

### Rationale

Downstream consumers depend on the data and its ordering, not exact bytes. Byte-matching would force replicating v1 quirks and abandon v2 gains.

### Alternatives Considered

- **Byte-for-byte equality**: Maximal safety for scripts parsing raw output - Rejected; impossible for HTML, forces replicating v1 array-join and styling quirks.

### Consequences

**Positive:**
- Allows adopting v2 improvements; realistic test bar.

**Negative:**
- Consumers parsing exact bytes (not just JSON/CSV fields) could see differences; mitigated by preserving key order and data for machine formats.

---

## Decision 4: Adopt v2's default responsive HTML template

**Date**: 2026-06-22
**Status**: accepted

### Context

v1's `html` format emitted a bare `<table>` fragment and (a known v1 quirk) wrote it to the file directly inside the renderer. v2 can render a full responsive HTML5 document via `DefaultHTMLTemplate`, written through the normal writer path.

### Decision

Use v2's default HTML template for the `html` format. The html output becomes a complete document rather than a fragment.

### Rationale

It is an improvement (standalone, responsive, styled) and removes the v1 direct-to-file quirk that the `file-output-format` spec called out. Consistent with the functional-equivalence bar (Decision 3), which allows html structure to change.

### Alternatives Considered

- **Minimal/fragment template**: Stays close to v1's bare table - Rejected; keeps awstools closer to a quirk being left behind and forgoes the responsive document with little benefit.
- **Defer to design phase**: Rejected; the choice is clear and unblocks requirements.

### Consequences

**Positive:**
- Standalone, styled HTML; file writing goes through the standard writer.

**Negative:**
- Anyone embedding the old bare `<table>` fragment must adapt; acceptable under the equivalence bar.

---

## Decision 5: Embrace v2 simplifications during the migration

**Date**: 2026-06-22
**Status**: accepted

### Context

v2's immutable Document renders to multiple writers in one pass, removing the need for v1 buffer workarounds. The migration can either swap APIs 1:1 or also remove those workarounds.

### Decision

Take the simplifications v2 enables: remove the `vpcenis` `AddToBuffer`/combined-array choreography, collapse `SeparateTables`/repeated-`Write()` into single multi-table documents, and let v2 auto-format multi-value cells where it cleanly replaces manual separator joins. Output must stay functionally equivalent (Decision 3).

### Rationale

The v1 workarounds exist only to fight v1's shared-buffer/Write-reset semantics, which no longer apply. Carrying them forward would preserve complexity for no behavioral reason.

### Alternatives Considered

- **Minimal 1:1 diff**: Smaller, lower-review-risk change - Rejected; leaves known awkward patterns in place and a larger amount of dead complexity.

### Consequences

**Positive:**
- Simpler output code; the most fragile v1-coupled file (`vpcenis`) is de-risked.

**Negative:**
- Slightly larger diff and more careful re-testing of the affected commands required.

---

## Decision 6: Remove the dead S3 output path

**Date**: 2026-06-22
**Status**: accepted

### Context

`names.go` and `organizations names` pass an S3 bucket to `format.PrintByteSlice`, but the bucket is read from a throwaway default settings object that no code populates, and no flag binds an S3 bucket/key. The S3 path is unreachable in practice.

### Decision

Remove the dead S3 output path rather than reimplement it on v2's `S3Writer`. The name-file commands write to stdout and `--file` only.

### Rationale

Wiring real S3 output is new functionality outside this migration's parity goal. Carrying a non-functional code path into v2 adds maintenance burden for no user benefit.

### Alternatives Considered

- **Wire S3 properly on v2 `S3Writer`**: Real feature - Rejected; out of scope, no flags or requirements exist for it.
- **Port the dead path as-is**: Rejected; preserves unreachable code.

### Consequences

**Positive:**
- Less dead code; clearer name-file output path.

**Negative:**
- If S3 output is wanted later, it is a new feature (acceptable; it never worked).

---

## Decision 7: Accept go-output v2's default CSV/table cell formatting (follow-up to review)

**Date**: 2026-06-23
**Status**: accepted

### Context

awstools hands raw `[]string` slices and `bool` values into output cells. Verified in both libraries' source: v1's CSV renderer joins slices with newlines within a cell and renders booleans as `Yes`/`No` (or ✅/❌ with `--emoji`); v2's CSV renderer formats a slice as Go's `[a b c]`, renders booleans as `true`/`false`, and strips embedded newlines/tabs to spaces. JSON and YAML are unaffected (both keep native arrays and booleans; the only JSON delta is v2 pretty-prints where v1 emitted compact JSON). So the migration's one real CSV/table regression is multi-value and boolean cell text.

This contradicted the original "let v2 auto-format multi-value cells" intent in [Decision 5](#decision-5-embrace-v2-simplifications-during-the-migration) only in that it has a visible CSV cost.

### Decision

Accept go-output v2's default cell formatting for CSV and table (`[a b c]` for lists, `true`/`false` for booleans). Do not build a v1-compatibility formatting layer as part of this migration. Open a follow-up task (to be filed in Transit) to review the CSV/table cell rendering after migration and restore v1-style cells (recoverable multi-value joins, `Yes`/`No` booleans) if it proves to matter to consumers.

### Rationale

The simpler migration ships sooner with less new code. The CSV change is visible but bounded (list and boolean columns only) and the machine-canonical formats (JSON/YAML) are unaffected, so most automated consumers are unaffected. Whether the CSV cell shape matters in practice is better judged after the migration against real output than guessed at now.

### Alternatives Considered

- **Preserve v1 CSV/table cell text (pre-join lists, `Yes`/`No` booleans)**: No CSV regression - Deferred, not rejected: captured as the follow-up task. Adds a per-cell formatting layer that is cheaper to scope once the v2 output is in hand.
- **Pre-join globally before handing to v2**: Simplest single code path - Rejected; it would also flatten JSON/YAML arrays into strings, losing v2's native-array benefit for the machine formats.

### Consequences

**Positive:**
- Smallest migration; JSON/YAML gain native array structure; no new formatting code now.

**Negative:**
- CSV multi-value cells become `[a b c]` and booleans become `true`/`false`; CSV consumers that parsed v1's newline-joined cells or `Yes`/`No` see a change. Tracked by the follow-up task.

### Follow-up

- **TODO (Transit):** "Review go-output v2 CSV/table cell rendering for multi-value and boolean columns; restore v1-style cells if needed." Not yet filed — Transit MCP was unavailable in the planning session.

---

## Decision 8: Central render helper lives in the config package

**Date**: 2026-07-03
**Status**: accepted

### Context

The v2 migration needs one place that builds `Output` instances (formats, writers, transformers) from viper configuration and renders a `Document` — replacing `config.NewOutputSettings()`. Candidate homes: the `config` package, package-level helpers in `cmd/`, or a new dedicated package.

### Decision

Implement `RenderDocument(ctx, doc)` and its supporting functions as methods on `*config.Config` in `config/config.go`.

### Rationale

`config` already owns every relevant viper key and the go-output import; commands already reach output configuration through the `settings` global (`cmd/root.go:15`). This is a direct replacement of `NewOutputSettings()` in the same home, preserving the existing architecture.

### Alternatives Considered

- **Helpers in `cmd/`**: Keeps config a thin viper wrapper - Rejected; splits output knowledge across two packages and leaks config keys into cmd.
- **New package (e.g. `internal/render`)**: Cleanest separation - Rejected; introduces an architectural layer the codebase doesn't have (CLAUDE.md: no new patterns without justification).

### Consequences

**Positive:**
- Smallest conceptual change for command authors; single home for output wiring.

**Negative:**
- `config` package grows beyond pure viper access (it already had output logic, so the drift is small).

---

## Decision 9: Convert commands from Run to RunE for error propagation

**Date**: 2026-07-03
**Status**: accepted

### Context

Requirement 9.1 demands render/write failures report an error and exit non-zero. v1's `Write()` returned nothing; commands use cobra `Run:` with scattered `panic(err)` / `log.Fatal(err)`. The v2 helper returns errors, so commands need a propagation style.

### Decision

Convert all migrated commands from `Run:` to `RunE:`; command bodies return errors (including replacing in-body `panic`/`log.Fatal` with `return err`). `rootCmd.Execute()` already exits non-zero on error.

### Rationale

RunE is the idiomatic cobra error contract: errors reach stderr and the exit code through one path, and command functions become testable without process exit. Chosen by the user over the smaller-diff alternative.

### Alternatives Considered

- **Keep `Run:` + a fatal wrapper (`renderOrFail`)**: Smallest diff, matches current style - Rejected by user in favor of the cleaner contract.

### Consequences

**Positive:**
- Uniform error handling; testable command bodies; satisfies R9.1 structurally.

**Negative:**
- Touches every command definition (signature churn) and slightly changes error output formatting (cobra prefixes `Error:`).

---

## Decision 10: graphEdges drops rows with nil to-cells instead of replicating v1's fmt artifact

**Date**: 2026-07-04
**Status**: accepted

### Context

v1's `OutputArray.splitFromToValues` stringified every to-cell with `toString`, so a missing/nil to-cell became the literal string `%!s(<nil>)`. Because that string is non-empty, v1's dot renderer emitted an edge from the row's from-node to an uninitialized zero-value node — garbage output triggered by rows like a VPC with no peerings. The new shared `graphEdges` helper (task 5, `cmd/graph.go`) had to either replicate this byte-for-byte or clean it up.

### Decision

`graphCellString` maps nil cells to the empty string, so rows without a to-cell produce no edges at all. All other v1 stringification semantics (comma join for `[]string`, split on `,`, empty-target skip) are replicated exactly and pinned by `TestGraphEdges`.

### Rationale

The v1 behavior is an artifact of `fmt.Sprintf("%s", nil)`, not intentional design: it rendered a bogus `%!s(<nil>)` node/edge in dot output. Reproducing a bug to preserve byte equivalence would make the graph output worse and would complicate the equivalence oracle tests (task 23), which would have to whitelist the artifact anyway.

### Alternatives Considered

- **Replicate `%!s(<nil>)` exactly**: Byte-for-byte v1 equivalence - Rejected because it preserves garbage output and a v1 rendering bug.
- **Emit the from-node without an edge**: Keeps lone nodes visible in graphs - Rejected because v2's Edge-based API has no isolated-node concept in this helper, and v1 only showed such nodes alongside the garbage edge; no command depends on it.

### Consequences

**Positive:**
- Graph output for rows without targets is clean (no phantom nodes/edges).
- Deviation is documented in code and pinned by a dedicated test.

**Negative:**
- dot/mermaid output is not byte-identical to v1 for rows with nil to-cells; the per-format equivalence tests (task 23) must account for this documented deviation.

---

## Decision 11: Accept go-output v2's JSON/YAML document envelope

**Date**: 2026-07-05
**Status**: accepted

### Context

v1 rendered `json`/`yaml` output as a bare top-level array of row objects, which downstream tooling consumed directly (e.g. `awstools s3 list | jq '.[]'`). go-output v2.7.0 always wraps table output in a document envelope — `{"title": ..., "data": [rows], "schema": {"keys": [...], "fields": [...]}}` — and exposes no data-only rendering option. The pre-push review found this divergence was shipped (and pinned by the per-format oracle tests) without being documented; requirement 2.1 and Decision 7 still claimed pretty-printing was the only JSON delta.

### Decision

Accept v2's document envelope for `json` and `yaml` output as a documented breaking change. Row data inside `data` keeps v1's fields, native value types, and key order; the envelope adds `title` and `schema` around it.

### Rationale

v2.7.0 offers no way to emit a bare array, so restoring v1's shape would require a custom Format or post-render transformer in the config package — permanent bespoke rendering code maintained solely to strip metadata that is also useful (the schema records column order, which raw JSON objects cannot express). The migration already accepts v2's rendering defaults for CSV cells (D7), HTML documents (D4), and the append marker (D3); the envelope is the same class of change. Downstream `jq` pipelines migrate mechanically (`.[]` → `.data[]`).

### Alternatives Considered

- **Custom unwrapping Format in config**: Restores byte-level v1 shape - Rejected because it adds permanent bespoke renderer code for all JSON/YAML output, contradicting the migration's goal of adopting v2's pipeline, and discards useful schema metadata.
- **Post-render transformer stripping the envelope**: Smaller code footprint - Rejected for the same maintenance reason; it would also break the file/stdout byte-equality invariant (R4.6) if applied to only one destination.

### Consequences

**Positive:**
- Zero custom rendering code; oracle tests pin v2's canonical shape.
- `schema.keys` preserves column order in JSON/YAML, which v1's bare arrays could not express.

**Negative:**
- Existing `jq`/YAML pipelines that index the top-level array break and must switch to `.data[]` (documented in README and CHANGELOG).
- Requirement 2.1's original strict-parity wording was amended after implementation rather than before — caught in review, not planning.

---
