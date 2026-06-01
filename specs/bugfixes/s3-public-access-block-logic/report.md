# Bugfix Report: S3 public-state logic misreads Public Access Block

**Date:** 2026-06-01
**Status:** Fixed
**Ticket:** T-1391

## Description of the Issue

`awstools s3 list` could misreport a bucket's public/private state ("Is Private").
`helpers/s3.go:computeBucketIsPublic` modelled the four S3 Public Access Block
(PAB) fields by their *write-prevention* effect instead of their *current-access*
effect.

**Reproduction steps:**
1. Create a bucket with a public bucket policy and a Public Access Block where
   `RestrictPublicBuckets=true` but `BlockPublicPolicy=false`.
2. Run `awstools s3 list`.
3. Observe the bucket is reported as public, although access granted by the
   public policy is actually restricted (effectively private).

A second variant: a bucket with a public ACL and `IgnorePublicAcls=true`,
`RestrictPublicBuckets=false` is reported public even though S3 ignores the ACL.

**Impact:** Incorrect security reporting for S3 buckets. Medium severity: the
tool is used to audit public exposure, so false positives (and the inverse risk
of false negatives) undermine its purpose.

## Investigation Summary

- **Symptoms examined:** Reported public/private state for buckets with partial
  PAB settings.
- **Code inspected:** `helpers/s3.go` (`computeBucketIsPublic`, lines ~283-332)
  and `helpers/s3_test.go` (`TestComputeBucketIsPublic`).
- **Hypotheses tested:** Confirmed against the AWS
  `PublicAccessBlockConfiguration` semantics that the function required the wrong
  combination of PAB fields to neutralise existing exposure.

## Discovered Root Cause

The function used write-prevention fields to neutralise existing access:

- A public **policy** was only neutralised when both `RestrictPublicBuckets` AND
  `BlockPublicPolicy` were true. `BlockPublicPolicy` only rejects *future*
  public bucket-policy writes; `RestrictPublicBuckets` is what restricts access
  granted by existing public policies.
- A public **ACL** was only neutralised when both `RestrictPublicBuckets` AND
  `IgnorePublicAcls` were true. `IgnorePublicAcls` alone causes S3 to ignore
  existing public ACLs.
- The "fully locked down" shortcut required `BlockPublicPolicy`, which is not the
  field that neutralises policy-based access.

**Defect type:** Logic error (wrong model of AWS PAB semantics).

**Why it occurred:** The fields were modelled by their write-prevention names
rather than their documented effect on current access.

**Contributing factors:** The four PAB fields have similar names and subtly
different effects, making the mismodelling easy to introduce.

## Resolution for the Issue

**Changes made:**
- `helpers/s3.go` (`computeBucketIsPublic`) - neutralise public **policy**
  exposure with `RestrictPublicBuckets` alone; neutralise public **ACL** exposure
  with `IgnorePublicAcls` alone; treat the bucket as fully locked down when
  `RestrictPublicBuckets && IgnorePublicAcls`; stop reading `BlockPublicPolicy`
  and `BlockPublicAcls`.

**Approach rationale:** Models the PAB fields by their effect on current access,
matching the AWS documentation. The tri-state ("unknown" = nil) behaviour from
prior tickets is preserved.

**Alternatives considered:**
- Keep reading the block-* fields as a secondary signal - rejected: they have no
  bearing on existing exposure and would re-introduce the bug.

## Regression Test

**Test file:** `helpers/s3_test.go`
**Test name:** `TestComputeBucketIsPublic`

**What it verifies:** Added four cases:
- `RestrictPublicBuckets=true, BlockPublicPolicy=false` + public policy -> not public.
- `IgnorePublicAcls=true, RestrictPublicBuckets=false` + public ACL -> not public.
- `BlockPublicPolicy=true, RestrictPublicBuckets=false` + public policy -> still public.
- `BlockPublicAcls=true, IgnorePublicAcls=false` + public ACL -> still public.

The two ticket-mandated cases fail against the pre-fix code (`got=true,
want=false`) and pass after the fix.

**Run command:** `go test ./helpers/ -run TestComputeBucketIsPublic -v`

## Affected Files

| File | Change |
|------|--------|
| `helpers/s3.go` | Corrected PAB modelling in `computeBucketIsPublic` |
| `helpers/s3_test.go` | Added regression and documentation test cases |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linters/validators pass (`go vet`, `gofmt`)

**Manual verification:**
- Confirmed red/green: regression cases fail pre-fix, pass post-fix.

## Prevention

**Recommendations to avoid similar bugs:**
- Model AWS feature flags by their documented runtime effect, not their names.
- Keep per-field table-driven tests that isolate each PAB field's effect.

## Related

- Ticket T-1391
