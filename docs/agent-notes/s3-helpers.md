# S3 helpers notes

Scope: `helpers/s3.go` plus its consumers in `cmd/s3list.go` and
`cmd/s3.go`.

## Architecture

- `GetBucketDetails(svc S3API) []S3Bucket` is the single entry point
  used by the CLI. It calls `GetAllBuckets` and then issues a
  sequence of per-bucket detail calls.
- `GetAllBuckets` walks every page of `ListBuckets` using the
  `ContinuationToken` on `ListBucketsInput`/`ListBucketsOutput`
  (T-835). AWS now paginates `ListBuckets` for accounts above the
  default 10k bucket quota, so a single call is no longer enough.
  Owner is captured from the first page (it's returned on every page
  but doesn't change).
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

## PublicAccessBlock tri-state (T-693)

`S3Bucket.PublicAccessBlockConfiguration` is a
`*types.PublicAccessBlockConfiguration` (pointer), not a value. A
`nil` pointer means the state is unknown — either the bucket has no
PAB configured (AWS returns `NoSuchPublicAccessBlockConfiguration`)
or the caller lacks `s3:GetBucketPublicAccessBlock`. This must be
distinct from a non-nil config whose four `*bool` flags are all
`false`, which is an explicit permissive configuration.

The renderer `parsePublicAccessBlock` in `cmd/s3list.go` returns the
literal string `"Unknown"` for the nil case. Previously the code
used a value type and silently swallowed the error, so unknown
buckets were rendered as "All false" — indistinguishable from the
legitimate all-false state.

## Error handling

`GetBucketDetails` logs a warning to `os.Stderr` for every failed
detail call via `warnS3DetailError`. It does not abort processing —
the failing field is simply left `nil`. `GetAllBuckets` still panics
on `ListBuckets` failure (pre-existing behaviour); there is no
useful fallback when the initial list cannot be obtained.

When adding a new per-bucket API call, prefer to represent the
unknown/absent state distinctly (pointer or explicit known flag)
rather than relying on zero values, to avoid collisions with
legitimate values.

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
- Regression tests for T-835 pagination:
  `TestGetAllBuckets_Pagination` (asserts every page is walked) and
  `TestGetAllBuckets_SinglePage` (asserts the loop stops after one
  call when no `ContinuationToken` is returned).
