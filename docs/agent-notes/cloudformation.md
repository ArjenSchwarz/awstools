# CloudFormation helpers

## Files

- `cmd/cfn.go` — parent `cfn` cobra command.
- `cmd/cfnresources.go` — `cfn resources` subcommand; lists resources in a
  stack and its nested stacks.
- `helpers/cfn.go` — pagination-aware wrappers around the CloudFormation
  listing APIs:
  - `GetResourcesByStackName(stackname, svc)` returns every resource for one
    stack by walking `NewListStackResourcesPaginator`. Panics on API error
    (legacy pattern; do not copy into new code).
  - `GetNestedCloudFormationResources(stackname, svc)` recursively expands
    `AWS::CloudFormation::Stack` entries by calling itself with the nested
    stack's `PhysicalResourceId`.
- `helpers/cfn_pagination_test.go` — regression tests for T-784 using a
  mock `cloudformation.ListStackResourcesAPIClient`.

## Why ListStackResources, not DescribeStackResources

The older `DescribeStackResources` API caps responses at 100 resources and
does not expose a `NextToken`, so it cannot be paginated. The AWS SDK's
`ListStackResources` exposes `NewListStackResourcesPaginator` and has no
such cap. We use it and convert each `StackResourceSummary` to the existing
`types.StackResource` shape so consumers (notably `cmd/cfnresources.go`)
stay unchanged. `StackName` is filled in from the input because summaries
do not repeat it per row (T-784).

## Testable interface pattern

The public helpers take `*cloudformation.Client`, but the actual work is
done in private functions that take `cloudformation.ListStackResourcesAPIClient`.
This matches the pattern used for TGW and ENI helpers and lets tests provide
a paginated mock without a real SDK client. See
`helpers/cfn_pagination_test.go` and `helpers/tgw_pagination_test.go` for
reference implementations of the mock.

## Gotchas

- `types.StackResource.PhysicalResourceId` is `*string` and can be nil. AWS
  returns nil while a resource is still in `CREATE_IN_PROGRESS` without a
  physical id yet, and for some resource types it stays nil. Always guard
  before dereferencing (T-733).
- `getNestedCloudFormationResources` skips nested-stack entries whose
  `PhysicalResourceId` is nil or empty. Without that guard, `ListStackResources`
  would treat a nil `StackName` as a reference to the parent and re-describe
  it, duplicating resources or causing runaway recursion.

## Testable conversion pattern

`cmd/cfnresources.go` exposes `buildCfnResource(resource, nameResolver)` as a
pure function so that per-resource conversion can be unit-tested without an
AWS client. Follow this pattern when adding similar commands.
