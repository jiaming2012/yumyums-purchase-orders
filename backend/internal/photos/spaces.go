package photos

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// SpacesConfig holds S3-compatible object storage credentials and bucket info.
// The store is Backblaze B2 since 2026-08 (previously DO Spaces — the type
// names keep the historical "Spaces" spelling; the wire protocol is plain S3).
type SpacesConfig struct {
	AccessKey string
	SecretKey string
	Endpoint  string // e.g. "https://s3.us-west-004.backblazeb2.com"
	Region    string // e.g. "us-west-004"
	Bucket    string
}

// NewSpacesClient creates an S3 client for the configured object store.
// Path-style addressing works on every S3-compatible provider we target.
func NewSpacesClient(cfg SpacesConfig) *s3.Client {
	return s3.New(s3.Options{
		Region:       cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		BaseEndpoint: aws.String(cfg.Endpoint),
		UsePathStyle: true,
	})
}

// NewSpacesPresigner creates a presign client for the configured object store.
func NewSpacesPresigner(cfg SpacesConfig) (*s3.PresignClient, error) {
	client := NewSpacesClient(cfg)
	return s3.NewPresignClient(client), nil
}

// GeneratePresignedPutURL generates a time-limited presigned PUT URL for uploading
// an object. No per-object ACL: Backblaze B2 rejects object ACLs that differ from
// the bucket's — public reads come from the bucket being public, not the object.
func GeneratePresignedPutURL(ctx context.Context, presigner *s3.PresignClient, bucket, key, contentType string, ttl time.Duration) (string, error) {
	req, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign PUT %s: %w", key, err)
	}
	return req.URL, nil
}

// GeneratePresignedGetURL generates a time-limited presigned GET URL for reading
// an object.
func GeneratePresignedGetURL(ctx context.Context, presigner *s3.PresignClient, bucket, key string, ttl time.Duration) (string, error) {
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign GET %s: %w", key, err)
	}
	return req.URL, nil
}

// PublicURL returns the permanent public URL for an object. Path-style, which
// every S3-compatible provider serves for public buckets.
// Format: https://{endpoint-host}/{bucket}/{key}
func PublicURL(endpoint, bucket, key string) string {
	// endpoint is like "https://s3.us-west-004.backblazeb2.com"
	return fmt.Sprintf("%s/%s/%s", endpoint, bucket, key)
}
