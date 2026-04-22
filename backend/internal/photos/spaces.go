package photos

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// SpacesConfig holds DO Spaces credentials and bucket info.
type SpacesConfig struct {
	AccessKey string
	SecretKey string
	Endpoint  string // e.g. "https://nyc3.digitaloceanspaces.com"
	Region    string // e.g. "nyc3"
	Bucket    string
}

// NewSpacesClient creates an S3 client configured for DO Spaces.
// DO Spaces requires path-style addressing (UsePathStyle: true).
func NewSpacesClient(cfg SpacesConfig) *s3.Client {
	return s3.New(s3.Options{
		Region:       cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		BaseEndpoint: aws.String(cfg.Endpoint),
		UsePathStyle: true,
	})
}

// NewSpacesPresigner creates a presign client configured for DO Spaces.
func NewSpacesPresigner(cfg SpacesConfig) (*s3.PresignClient, error) {
	client := NewSpacesClient(cfg)
	return s3.NewPresignClient(client), nil
}

// GeneratePresignedPutURL generates a time-limited presigned PUT URL for uploading
// an object to DO Spaces.
func GeneratePresignedPutURL(ctx context.Context, presigner *s3.PresignClient, bucket, key, contentType string, ttl time.Duration) (string, error) {
	req, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		ACL:         "public-read",
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign PUT %s: %w", key, err)
	}
	return req.URL, nil
}

// GeneratePresignedGetURL generates a time-limited presigned GET URL for reading
// an object from DO Spaces.
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

// PublicURL returns the permanent public URL for an object in DO Spaces.
// Uses path-style URLs to support bucket names with dots (e.g. "hq.yumyums")
// which break subdomain-style due to wildcard SSL cert limitations.
// Format: https://{region}.digitaloceanspaces.com/{bucket}/{key}
func PublicURL(endpoint, bucket, key string) string {
	// endpoint is like "https://nyc3.digitaloceanspaces.com"
	return fmt.Sprintf("%s/%s/%s", endpoint, bucket, key)
}
