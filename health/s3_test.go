package health

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockS3Client is a mock implementation of the s3iface.S3API interface.
type MockS3Client struct {
	s3iface.S3API
	mock.Mock
}

func (m *MockS3Client) HeadBucketWithContext(ctx aws.Context, input *s3.HeadBucketInput, opts ...request.Option) (*s3.HeadBucketOutput, error) {
	args := m.Called(ctx, input, opts)
	return args.Get(0).(*s3.HeadBucketOutput), args.Error(1)
}

func TestS3Checker_Check_Success(t *testing.T) {
	mockS3 := new(MockS3Client)

	mockS3.On("HeadBucketWithContext", mock.Anything, &s3.HeadBucketInput{
		Bucket: aws.String("test-bucket"),
	}, mock.Anything).Return(&s3.HeadBucketOutput{}, nil)

	config := S3CheckerConfig{
		BucketName: "test-bucket",
		Name:       "test-s3-check",
		Region:     "us-east-1",
		S3Client:   mockS3,
	}
	checker, _ := NewS3Checker(config)

	err := checker.Check(context.Background())
	assert.NoError(t, err)

	mockS3.AssertExpectations(t)
}

func TestS3Checker_Check_OtherAWSError(t *testing.T) {
	mockS3 := new(MockS3Client)
	awsErr := awserr.New("SomeAWSError", "some aws error", nil)
	mockS3.On("HeadBucketWithContext", mock.Anything, mock.Anything, mock.Anything).Return((*s3.HeadBucketOutput)(nil), awsErr)

	config := S3CheckerConfig{S3Client: mockS3, BucketName: "test", Name: "test"}
	checker, _ := NewS3Checker(config)
	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "s3 HeadBucket failed")
	assert.ErrorIs(t, err, awsErr)
	mockS3.AssertExpectations(t)
}

func TestS3Checker_NewS3Checker_SessionCreationError(t *testing.T) {
	config := S3CheckerConfig{
		BucketName: "test-bucket",
		Name:       "test-s3-check",
		Region:     "us-east-1",
	}

	_, err := NewS3Checker(config)

	assert.Error(t, err)
	assert.EqualError(t, err, "failed to create AWS session: no S3 interface detected")
}

func TestS3Checker_Name(t *testing.T) {
	mockS3 := new(MockS3Client)
	config := S3CheckerConfig{Name: "test-s3-check", BucketName: "test", S3Client: mockS3}
	checker, err := NewS3Checker(config)
	if err != nil {
		t.Fatalf("NewS3Checker returned an unexpected error: %v", err)
	}
	assert.Equal(t, "test-s3-check", checker.Name())
}

func TestS3Checker_Type(t *testing.T) {
	mockS3 := new(MockS3Client)
	config := S3CheckerConfig{Name: "test", BucketName: "test", S3Client: mockS3}
	checker, err := NewS3Checker(config)
	if err != nil {
		t.Fatalf("NewS3Checker returned an unexpected error: %v", err)
	}
	assert.Equal(t, "s3", checker.Type())
}

func TestS3Checker_Check_ContextCancelled_DuringRequest(t *testing.T) {
	mockS3 := new(MockS3Client)
	mockS3.On("HeadBucketWithContext", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			time.Sleep(200 * time.Millisecond)
		}).
		Return((*s3.HeadBucketOutput)(nil), context.DeadlineExceeded)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	config := S3CheckerConfig{S3Client: mockS3, BucketName: "test", Name: "Test"}
	checker, _ := NewS3Checker(config)

	err := checker.Check(ctx)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "s3 HeadBucket failed")
	mockS3.AssertExpectations(t)
}
