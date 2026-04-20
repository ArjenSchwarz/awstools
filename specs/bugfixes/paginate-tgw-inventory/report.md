# Bugfix Report: Paginate Transit Gateway Inventory Helpers

**Date:** 2026-04-20
**Status:** Fixed
**Ticket:** T-669

## Description of the Issue

Several Transit Gateway inventory helpers in `helpers/ec2.go` made single-page
AWS API calls and silently dropped any resources that spilled into subsequent
pages. Accounts with many TGWs, route tables, or associations would see
incomplete output from:

- `awstools tgw overview`
- `awstools tgw routes`
- `awstools tgw dangling`

Affected helpers:

- `GetAllTransitGateways` — calls `DescribeTransitGateways` once.
- `GetRouteTablesForTransitGateway` — calls `DescribeTransitGatewayRouteTables` once.
- `GetSourceAttachmentsForTransitGatewayRouteTable` — calls `GetTransitGatewayRouteTableAssociations` once.
- `GetActiveRoutesForTransitGatewayRouteTable` / `GetBlackholeRoutesForTransitGatewayRouteTable` — call `SearchTransitGatewayRoutes` once (capped at 1000 results with no `NextToken`).

**Reproduction steps:**

1. Run `awstools tgw overview` against an account with more than one page of
   TGWs (or TGW route tables, or associations).
2. Observe that only the first page's worth of resources is returned.

**Impact:** Medium. Inventory commands silently under-report resources in
large environments, which can lead to missed dangling routes or missing
entries in diagrams.

## Investigation Summary

Followed the same pattern used in previous pagination fixes
(`getAllVPCRouteTables`, `GetNetworkInterfaces`, IAM pagination).

- **Symptoms examined:** Ticket T-669 description — TGW inventory helpers
  miss resources in large environments.
- **Code inspected:** `helpers/ec2.go` — the five TGW helpers listed above
  and their callers in `cmd/tgwoverview.go`, `cmd/tgwroutes.go`,
  `cmd/tgwdangling.go`.
- **Hypotheses tested:** Confirmed the SDK provides
  `NewDescribeTransitGatewaysPaginator`,
  `NewDescribeTransitGatewayRouteTablesPaginator`, and
  `NewGetTransitGatewayRouteTableAssociationsPaginator`. Confirmed that
  `SearchTransitGatewayRoutes` has no SDK paginator because the AWS API
  caps results at 1000 with only an `AdditionalRoutesAvailable` flag.

## Discovered Root Cause

Three of the helpers build their AWS input and then call the one-shot API
method instead of walking an SDK paginator. The fourth (route search) has
no pagination token available in the API, so it silently truncates at 1000
routes.

**Defect type:** Missing pagination — API results truncated at the first
page boundary.

**Why it occurred:** Pagination was not applied when these helpers were
first written. Unlike the VPC route-table and ENI helpers, nothing prompted
a refactor to the paginator pattern.

**Contributing factors:** `SearchTransitGatewayRoutes` is an unusual AWS
API in that it has no `NextToken`. The only way to work around the 1000-
result cap is to narrow the filter set.

## Resolution for the Issue

**Changes made:**

- `helpers/ec2.go` — `GetAllTransitGateways` now delegates to
  `getAllTransitGateways`, which takes the narrow
  `ec2.DescribeTransitGatewaysAPIClient` interface and walks
  `NewDescribeTransitGatewaysPaginator`.
- `helpers/ec2.go` — `GetRouteTablesForTransitGateway` now delegates to
  `getRouteTablesForTransitGateway`, which takes a composite TGW APIClient
  interface and walks `NewDescribeTransitGatewayRouteTablesPaginator`.
- `helpers/ec2.go` — `GetSourceAttachmentsForTransitGatewayRouteTable` now
  delegates to `getSourceAttachmentsForTransitGatewayRouteTable`, which
  walks `NewGetTransitGatewayRouteTableAssociationsPaginator`.
- `helpers/ec2.go` — `GetActiveRoutesForTransitGatewayRouteTable` now sets
  `MaxResults: 1000` explicitly and, when
  `AdditionalRoutesAvailable` is true, re-queries per route type
  (`propagated` and `static`) to work around the API's 1000-result cap.
- `helpers/ec2.go` — `GetBlackholeRoutesForTransitGatewayRouteTable` now
  sets `MaxResults: 1000` explicitly.
- `helpers/tgw_pagination_test.go` — new regression tests covering all of
  the above, following the `APIClient` mock pattern used by
  `helpers/vpc_routetable_pagination_test.go` and
  `helpers/ec2_pagination_test.go`.

**Approach rationale:** Mirrors the existing codebase patterns. Keeping a
public wrapper that takes `*ec2.Client` avoids a breaking change for
callers; the private implementation takes the narrow APIClient interface
so tests can mock it.

**Alternatives considered:**

- Rewriting the public API to take the interface directly — rejected to
  keep the diff minimal and avoid touching every caller.
- For `SearchTransitGatewayRoutes`, ignoring the overflow and documenting
  the 1000-route cap — rejected because the ticket explicitly calls out
  route searches as affected and the split-by-type fallback raises the
  effective ceiling to 2000 active routes per route table at zero
  additional cost when routes are below 1000.

## Regression Test

**Test file:** `helpers/tgw_pagination_test.go`

**Test names:**

- `TestGetAllTransitGateways_Pagination`
- `TestGetRouteTablesForTransitGateway_Pagination`
- `TestGetSourceAttachmentsForTransitGatewayRouteTable_Pagination`
- `TestGetActiveRoutesForTransitGatewayRouteTable_SplitsOnOverflow`
- `TestGetBlackholeRoutesForTransitGatewayRouteTable_UsesMaxResults`

**What they verify:**

- Each helper walks all pages of the mock AWS API and returns the complete
  set of resources, not just the first page.
- The active-route helper splits its search by route type when the API
  signals more results are available.

**Run command:** `go test ./helpers/ -run 'TransitGateway|Tgw'`

## Affected Files

| File | Change |
|------|--------|
| `helpers/ec2.go` | Refactor TGW inventory helpers to use SDK paginators and the APIClient pattern; add split-by-type fallback for route searches. |
| `helpers/tgw_pagination_test.go` | New pagination regression tests. |
| `docs/agent-notes/ec2-helpers.md` | Document the TGW pagination pattern. |

## Verification

**Automated:**

- [x] Regression tests pass
- [x] Full test suite passes
- [x] `go fmt ./...` clean
- [x] `go vet ./...` clean

**Manual verification:**

- Inspected call sites in `cmd/tgwoverview.go`, `cmd/tgwroutes.go`,
  `cmd/tgwdangling.go` — no signature changes required.

## Prevention

**Recommendations to avoid similar bugs:**

- Whenever a helper calls an AWS `Describe…` / `List…` / `Get…` API, check
  whether the SDK exposes a paginator for it. If so, use it.
- Add pagination tests for any new listing helper.
- Be aware that `SearchTransitGatewayRoutes` is an exception — it has no
  `NextToken` and must be worked around with narrower filters.

## Related

- Similar pattern: `helpers/vpc_routetable_pagination_test.go` (VPC route
  tables), `helpers/ec2_pagination_test.go` (ENIs).
