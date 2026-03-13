# Bugfix Report: s3-acl-nil-grantee-uri

**Date:** 2026-03-13
**Status:** Fixed
**Transit Ticket:** T-338

## Description of the Issue

The `GetBucketDetails` function in `helpers/s3.go` panics with a nil pointer dereference when processing S3 bucket ACL grants that have a nil `Grantee.URI` field. This occurs for grant types like `TypeAmazonCustomerByEmail`, where `EmailAddress` is set but `URI` is nil.

**Reproduction steps:**
1. Have an S3 bucket with a grant of type `AmazonCustomerByEmail` (email-based ACL)
2. Run any `s3 list` command that calls `GetBucketDetails`
3. The program panics on `*acl.Grantee.URI` dereference

**Impact:** Any AWS account with email-based S3 grants cannot use the `s3 list` commands — the entire scan aborts.

## Investigation Summary

- **Symptoms examined:** Panic on `*acl.Grantee.URI` at line 70 of `helpers/s3.go`
- **Code inspected:** `GetBucketDetails` ACL processing loop in `helpers/s3.go`
- **Hypotheses tested:** The `URI` field is only populated for `TypeGroup` grantees. For `TypeAmazonCustomerByEmail` and potentially other types, `URI` is nil.

## Discovered Root Cause

**Defect type:** Missing nil guard (nil pointer dereference)

**Why it occurred:** The ACL loop checked `acl.Grantee.Type != types.TypeCanonicalUser` and then immediately dereferenced `*acl.Grantee.URI`. This assumes all non-canonical-user grantees have a URI, but `TypeAmazonCustomerByEmail` grantees use `EmailAddress` instead.

**Contributing factors:** The AWS S3 SDK uses pointer fields for optional values. The `Grantee.URI` field is only set for group-type grantees, not for email-based grantees.

## Resolution for the Issue

**Changes made:**
- `helpers/s3.go` - Extracted inline ACL loop into `hasOpenACLs()` function with nil-safe URI handling via `aws.ToString()`
- `helpers/s3_test.go` - Added `TestHasOpenACLs_NilGranteeURI` with 7 test cases covering nil URI, nil Grantee, and all grant types

**Approach rationale:** Used `aws.ToString()` which safely returns `""` for nil `*string` pointers, consistent with the rest of the codebase. Extracted the logic to a named function to make it independently testable.

**Alternatives considered:**
- Inline nil check (`if acl.Grantee.URI != nil && ...`) — less testable, kept the complex condition inline
- Type-switch on `Grantee.Type` — more verbose, and not all types are known ahead of time

## Regression Test

**Test file:** `helpers/s3_test.go`
**Test name:** `TestHasOpenACLs_NilGranteeURI`

**What it verifies:** That `hasOpenACLs` handles nil `Grantee.URI` (email grants), nil `Grantee` pointers, log delivery groups, public groups, and mixed grant lists without panicking and with correct boolean results.

**Run command:** `go test ./helpers/ -run TestHasOpenACLs -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/s3.go` | Extracted `hasOpenACLs()` with nil-safe URI handling |
| `helpers/s3_test.go` | Added regression test `TestHasOpenACLs_NilGranteeURI` |

## Verification

**Automated:**
- [x] Regression test passes (7/7 subtests)
- [x] Full test suite passes
- [x] `go fmt` passes

## Prevention

**Recommendations to avoid similar bugs:**
- Always use `aws.ToString()` instead of raw `*ptr` dereference for optional SDK string fields
- Extract complex conditional logic into named functions for testability
- Consider adding a linter rule for raw pointer dereferences on AWS SDK types

## Related

- Transit ticket: T-338
