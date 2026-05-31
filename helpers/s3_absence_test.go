package helpers

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
)

// s3APIErr builds a smithy API error with the given code, matching what
// the AWS S3 SDK surfaces for absence states such as NoSuchTagSet.
func s3APIErr(code string) error {
	return &smithy.GenericAPIError{Code: code, Message: code}
}

// TestIsS3AbsenceError verifies the error classifier matches known
// absence codes via smithy.APIError and rejects everything else.
func TestIsS3AbsenceError(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		codes []string
		want  bool
	}{
		{
			name:  "matching single code",
			err:   s3APIErr(s3ErrNoSuchTagSet),
			codes: []string{s3ErrNoSuchTagSet},
			want:  true,
		},
		{
			name:  "matching one of several codes",
			err:   s3APIErr(s3ErrNoSuchBucketPolicy),
			codes: []string{s3ErrNoSuchTagSet, s3ErrNoSuchBucketPolicy},
			want:  true,
		},
		{
			name:  "non-matching API error code",
			err:   s3APIErr("AccessDenied"),
			codes: []string{s3ErrNoSuchTagSet},
			want:  false,
		},
		{
			name:  "non-API (transport) error",
			err:   errBoomAbsence{},
			codes: []string{s3ErrNoSuchTagSet},
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isS3AbsenceError(tt.err, tt.codes...); got != tt.want {
				t.Errorf("isS3AbsenceError() = %v, want %v", got, tt.want)
			}
		})
	}
}

type errBoomAbsence struct{}

func (errBoomAbsence) Error() string { return "connection refused" }

// TestGetBucketDetails_AbsenceErrorsAreDefiniteState is the regression
// test for T-1093. When S3 reports a known "configuration absent" error
// code, the corresponding state must be a definite empty/false value
// (not "unknown"), and no warning behaviour should apply.
func TestGetBucketDetails_AbsenceErrorsAreDefiniteState(t *testing.T) {
	const bucketName = "absent-config-bucket"

	t.Run("encryption absent => HasEncryption false", func(t *testing.T) {
		silenceStderr(t)
		mock := healthyS3Mock(bucketName)
		mock.getBucketEncryption = func(ctx context.Context, params *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error) {
			return nil, s3APIErr(s3ErrServerSideEncryptionConfigurationNotFound)
		}

		buckets := GetBucketDetails(mock)
		if len(buckets) != 1 {
			t.Fatalf("expected 1 bucket, got %d", len(buckets))
		}
		b := buckets[0]
		if b.HasEncryption == nil {
			t.Fatal("HasEncryption is nil (unknown), want definite false")
		}
		if *b.HasEncryption {
			t.Errorf("HasEncryption = true, want false")
		}
	})

	t.Run("tags absent => Tags empty, no warning path", func(t *testing.T) {
		silenceStderr(t)
		mock := healthyS3Mock(bucketName)
		mock.getBucketTagging = func(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
			return nil, s3APIErr(s3ErrNoSuchTagSet)
		}

		buckets := GetBucketDetails(mock)
		if len(buckets) != 1 {
			t.Fatalf("expected 1 bucket, got %d", len(buckets))
		}
		if len(buckets[0].Tags) != 0 {
			t.Errorf("Tags = %v, want empty", buckets[0].Tags)
		}
	})

	t.Run("policy absent => Policy empty", func(t *testing.T) {
		silenceStderr(t)
		mock := healthyS3Mock(bucketName)
		mock.getBucketPolicy = func(ctx context.Context, params *s3.GetBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyOutput, error) {
			return nil, s3APIErr(s3ErrNoSuchBucketPolicy)
		}

		buckets := GetBucketDetails(mock)
		if len(buckets) != 1 {
			t.Fatalf("expected 1 bucket, got %d", len(buckets))
		}
		if buckets[0].Policy != "" {
			t.Errorf("Policy = %q, want empty", buckets[0].Policy)
		}
	})

	t.Run("replication absent => no rules", func(t *testing.T) {
		silenceStderr(t)
		mock := healthyS3Mock(bucketName)
		mock.getBucketReplication = func(ctx context.Context, params *s3.GetBucketReplicationInput, optFns ...func(*s3.Options)) (*s3.GetBucketReplicationOutput, error) {
			return nil, s3APIErr(s3ErrReplicationConfigurationNotFound)
		}

		buckets := GetBucketDetails(mock)
		if len(buckets) != 1 {
			t.Fatalf("expected 1 bucket, got %d", len(buckets))
		}
		if len(buckets[0].Replication.Rules) != 0 {
			t.Errorf("Replication.Rules = %v, want none", buckets[0].Replication.Rules)
		}
	})
}

// TestGetBucketDetails_RealEncryptionErrorStaysUnknown verifies that a
// genuine (non-absence) error on the encryption call still leaves
// HasEncryption nil so the renderer reports it as Unknown.
func TestGetBucketDetails_RealEncryptionErrorStaysUnknown(t *testing.T) {
	silenceStderr(t)
	mock := healthyS3Mock("real-error-bucket")
	mock.getBucketEncryption = func(ctx context.Context, params *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error) {
		return nil, s3APIErr("AccessDenied")
	}

	buckets := GetBucketDetails(mock)
	if len(buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(buckets))
	}
	if buckets[0].HasEncryption != nil {
		t.Errorf("HasEncryption = %v, want nil (unknown) on a real error", *buckets[0].HasEncryption)
	}
}
