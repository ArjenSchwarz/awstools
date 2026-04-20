# S3 helpers notes

Scope: `helpers/s3.go` plus its consumers in `cmd/s3list.go` and
`cmd/s3.go`.

## Architecture

- `GetBucketDetails(svc S3API) []S3Bucket` is the single entry point
  used by the CLI. It calls `ListBuckets` once and then issues a
  sequence of per-bucket detail calls.
- `S3API` is a helper-owned interface (added for T-714). It lists
  only the S3 SDK methods used inside the helper so tests can inject
  per-call failures with a `mockS3Client`.
- The production `*s3.Client` satisfies `S3API` implicitly — no
  change needed at the call site beyond the signature.

## Tri-state booleans (T-714)

Several fields on `S3Bucket` are `*bool` rather than `bool`:

- `IsPublic`, `PublicPolicy`, `OpenACLs`
- `LoggingEnabled`
- `HasEncryption`
- `Versioning`, `VersioningMFAEnabled`

A `nil` pointer means the underlying AWS call failed and the state
is unknown. A non-nil pointer is a confirmed answer. Never treat
`nil` as "off" for security decisions — it must stay explicit. In
particular:

- `cmd/s3list.go` filters (`--public-only`, `--unencrypted-only`)
  exclude confirmed-safe buckets but keep unknowns visible so the
  user can investigate.
- `cmd/s3list.go` render helpers (`triState`, `negatedTriState`)
  print `Unknown` for nil values.

`IsPublic` is a composite of policy status + ACL state + PAB. The
aggregation lives in `computeBucketIsPublic`. Rules:

- A confirmed-public input makes the bucket public unless the PAB
  fully neutralises it.
- A fully-locked PAB (`Restrict + BlockPolicy + IgnorePublicAcls`
  all true) forces "not public" even when inputs are unknown.
- Otherwise, any unknown input makes the composite unknown too,
  because the unknown side could flip the answer.

## PAB handling is T-693's scope

`PublicAccessBlockConfiguration` itself is still the SDK type (a
value, not a pointer). T-693 handles the "unknown PAB" case
separately. T-714 intentionally left it alone.

## Error handling

`GetBucketDetails` logs a warning to `os.Stderr` for every failed
detail call via `warnS3DetailError`. It does not abort processing —
the failing field is simply left `nil`. `GetAllBuckets` still panics
on `ListBuckets` failure (pre-existing behaviour); there is no
useful fallback when the initial list cannot be obtained.

## Testing

- `helpers/s3_test.go` has `mockS3Client` (implements `S3API`) plus
  `healthyS3Mock()` which returns a mock where every call succeeds
  with benign data. Tests override individual function fields to
  simulate specific failures.
- Use `silenceStderr(t)` to suppress the warning noise that
  `GetBucketDetails` writes on failure paths.
- Regression tests for T-714:
  `TestGetBucketDetails_UnknownOnDetailErrors`,
  `TestGetBucketDetails_HealthyPathSetsPointers`,
  `TestComputeBucketIsPublic`.
