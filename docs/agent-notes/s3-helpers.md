# S3 Helpers

## PublicAccessBlock tri-state

`S3Bucket.PublicAccessBlockConfiguration` is a `*types.PublicAccessBlockConfiguration` (pointer), not a value. A `nil` pointer means the state is unknown — either the bucket has no PAB configured (AWS returns `NoSuchPublicAccessBlockConfiguration`) or the caller lacks `s3:GetBucketPublicAccessBlock`. This must be distinct from a non-nil config whose four `*bool` flags are all `false`, which is an explicit permissive configuration.

The renderer `parsePublicAccessBlock` in `cmd/s3list.go` returns the literal string `"Unknown"` for the nil case. Previously the code used a value type and silently swallowed the error, so unknown buckets were rendered as "All false" — indistinguishable from the legitimate all-false state (bug T-693).

## GetBucketDetails error tolerance

`GetBucketDetails` tolerates most per-bucket API errors (PAB, policy, ACL, logging, encryption, tags, replication, versioning) because AWS frequently returns "NoSuch*" errors when a feature is not configured. When adding a new per-bucket API call, prefer to represent the unknown/absent state distinctly (pointer or explicit known flag) rather than relying on zero values, to avoid collisions with legitimate values.
