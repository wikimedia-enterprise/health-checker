package health

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// S3CheckerConfig holds configuration for the S3Checker.
type S3CheckerConfig struct {
	BucketName string
	Name       string
	Region     string
	S3Client   s3iface.S3API
}

// S3Checker implements the HealthChecker interface for AWS S3.
type S3Checker struct {
	config   S3CheckerConfig
	s3Client s3iface.S3API
}

// NewS3Checker creates a new S3Checker.  Handles optional client injection.
func NewS3Checker(config S3CheckerConfig) (*S3Checker, error) {
	var client s3iface.S3API
	if config.S3Client != nil {
		client = config.S3Client
	} else {
		return nil, fmt.Errorf("failed to create AWS session: %w", fmt.Errorf("no S3 interface detected"))
	}

	return &S3Checker{config: config, s3Client: client}, nil
}

// Check performs the S3 health check.
func (c *S3Checker) Check(ctx context.Context) error {
	_, err := c.s3Client.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.config.BucketName),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, "NotFound":
				return fmt.Errorf("s3 bucket %q does not exist or is not accessible: %w", c.config.BucketName, err)
			case "AccessDenied":
				return fmt.Errorf("access denied to s3 bucket %q: %w", c.config.BucketName, err)
			default:
				return fmt.Errorf("s3 HeadBucket failed: %w", err)

			}
		}
		return fmt.Errorf("s3 HeadBucket failed: %w", err)
	}

	return nil
}

// Name returns the name of the health check.
func (c *S3Checker) Name() string {
	return c.config.Name
}

// Type returns the type of the health check (s3).
func (c *S3Checker) Type() string {
	return "s3"
}
