# CloudFormation helpers

## Files

- `cmd/cfn.go` — parent `cfn` cobra command.
- `cmd/cfnresources.go` — `cfn resources` subcommand; lists resources in a
  stack and its nested stacks.
- `helpers/cfn.go` — thin wrappers around `DescribeStackResources`:
  - `GetResourcesByStackName(stackname, svc)` returns the resources for one
    stack. Panics on API error (legacy pattern; do not copy into new code).
  - `GetNestedCloudFormationResources(stackname, svc)` recursively expands
    `AWS::CloudFormation::Stack` entries by calling itself with the nested
    stack's `PhysicalResourceId`.

## Gotchas

- `types.StackResource.PhysicalResourceId` is `*string` and can be nil. AWS
  returns nil while a resource is still in `CREATE_IN_PROGRESS` without a
  physical id yet, and for some resource types it stays nil. Always guard
  before dereferencing (T-733).
- `GetNestedCloudFormationResources` passes `resource.PhysicalResourceId`
  directly into the recursive call. For `AWS::CloudFormation::Stack`
  resources this is usually set once the nested stack has been created, but
  if it is nil the recursive `DescribeStackResources` call would re-describe
  the top-level stack (since `StackName` defaults to the current stack when
  the pointer is nil). Worth tightening if a regression is reported.

## Testable conversion pattern

`cmd/cfnresources.go` exposes `buildCfnResource(resource, nameResolver)` as a
pure function so that per-resource conversion can be unit-tested without an
AWS client. Follow this pattern when adding similar commands.
