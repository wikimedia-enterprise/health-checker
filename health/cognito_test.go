package health

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider/cognitoidentityprovideriface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCognitoClient is a mock implementation of the cognitoidentityprovideriface.CognitoIdentityProviderAPI interface.
type MockCognitoClient struct {
	cognitoidentityprovideriface.CognitoIdentityProviderAPI
	mock.Mock
}

func (m *MockCognitoClient) DescribeUserPoolClientWithContext(ctx aws.Context, input *cognitoidentityprovider.DescribeUserPoolClientInput, opts ...request.Option) (*cognitoidentityprovider.DescribeUserPoolClientOutput, error) {
	args := m.Called(ctx, input, opts)
	return args.Get(0).(*cognitoidentityprovider.DescribeUserPoolClientOutput), args.Error(1)
}

func TestCognitoChecker_Check_Success(t *testing.T) {
	mockCognito := new(MockCognitoClient)

	mockCognito.On("DescribeUserPoolClientWithContext", mock.Anything, &cognitoidentityprovider.DescribeUserPoolClientInput{
		UserPoolId: aws.String("pool-id"),
		ClientId:   aws.String("client-id"),
	}, mock.Anything).Return(&cognitoidentityprovider.DescribeUserPoolClientOutput{}, nil)

	checker := CognitoChecker{
		Timeout:         1 * time.Second,
		CheckerName:     "test-cognito-check",
		CognitoAPI:      mockCognito,
		UserPoolId:      "pool-id",
		CognitoClientID: "client-id",
	}

	err := checker.Check(context.Background())
	assert.NoError(t, err)

	mockCognito.AssertExpectations(t)
}

func TestCognitoChecker_Check_OtherAWSError(t *testing.T) {
	mockCognito := new(MockCognitoClient)
	awsErr := awserr.New("SomeAWSError", "some aws error", nil)
	mockCognito.On("DescribeUserPoolClientWithContext", mock.Anything, mock.Anything, mock.Anything).Return((*cognitoidentityprovider.DescribeUserPoolClientOutput)(nil), awsErr)

	checker := CognitoChecker{CognitoAPI: mockCognito, CheckerName: "test"}
	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DescribeUserPoolClientWithContext failed")
	assert.ErrorIs(t, err, awsErr)
	mockCognito.AssertExpectations(t)
}

func TestCognitoChecker_Getters(t *testing.T) {
	mockCognito := new(MockCognitoClient)
	checker := CognitoChecker{
		CheckerName: "test-cognito-check",
		CognitoAPI:  mockCognito,
		Timeout:     5 * time.Second,
	}
	assert.Equal(t, "test-cognito-check", checker.Name())
	assert.Equal(t, "cognito", checker.Type())
	assert.Equal(t, 5*time.Second, checker.GetTimeOut())
}

func TestCognitoChecker_Check_ContextCancelled_DuringRequest(t *testing.T) {
	mockCognito := new(MockCognitoClient)
	mockCognito.On("DescribeUserPoolClientWithContext", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			time.Sleep(200 * time.Millisecond)
		}).
		Return((*cognitoidentityprovider.DescribeUserPoolClientOutput)(nil), context.DeadlineExceeded)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	checker := CognitoChecker{CognitoAPI: mockCognito, UserPoolId: "test", CheckerName: "Test", Timeout: 5 * time.Second}

	err := checker.Check(ctx)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "DescribeUserPoolClientWithContext failed")
	mockCognito.AssertExpectations(t)
}
