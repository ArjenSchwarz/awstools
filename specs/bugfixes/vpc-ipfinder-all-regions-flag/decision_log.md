# Decision Log: vpc-ipfinder-all-regions-flag

## Decision 1: Reject --search-all-regions instead of implementing multi-region search

**Date**: 2026-06-01
**Status**: accepted

### Context

`cmd/vpcipfinder.go` registered a `--search-all-regions` flag bound to the
`searchAllRegions` variable, but the variable was never read. The command always
searched only the current region, so passing the flag produced a silent no-op
and could give users a false "not found" result. T-1222 asks to either implement
the multi-region search or remove/disable the flag, choosing the simpler correct
option and documenting the choice.

### Decision

Reject the `--search-all-regions` flag with a clear error until multi-region
search is implemented, rather than implementing the multi-region search now. The
flag remains registered (for discoverability and forward compatibility) but
returns an explanatory error when used. Help text and the command long
description are updated to state the search is single-region.

### Rationale

Implementing a correct multi-region search is a non-trivial change: enumerate
all enabled regions, create a per-region EC2 client, run the existing lookup in
each, aggregate and de-duplicate results, and handle partial failures and the
extra API cost. That is well beyond a bug fix in scope and risk. Rejecting the
flag is small, removes the false-negative immediately, and gives the user an
actionable message (use `--region`). The flag's own help text already labelled
it a "future enhancement", confirming the feature was never delivered.

### Alternatives Considered

- **Implement multi-region search**: Make the flag actually search every region
  and aggregate results - Rejected for this ticket because it is a sizeable
  feature (region enumeration, per-region clients, aggregation, error handling,
  AWS-call cost) that exceeds the scope and risk appropriate for a bug fix.
- **Remove the flag entirely**: Delete the flag registration and variable -
  Rejected because an unknown-flag error is less helpful than an explicit "not
  yet supported" message, and keeping the flag preserves discoverability and a
  stable surface for when the feature lands.

### Consequences

**Positive:**
- No more silent no-op; users get a clear, actionable error.
- Flag validation is extracted into a pure function and unit tested without AWS
  credentials.
- The path to a real implementation stays open (flag already exists).

**Negative:**
- The flag still cannot perform a multi-region search; that work is deferred.
- A user who scripted around the (previously ignored) flag will now get an error
  instead of single-region results. This is the intended correction.

---
