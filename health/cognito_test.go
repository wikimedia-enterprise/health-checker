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

func (m *MockCognitoClient) InitiateAuthWithContext(ctx aws.Context, input *cognitoidentityprovider.InitiateAuthInput, opts ...request.Option) (*cognitoidentityprovider.InitiateAuthOutput, error) {
	args := m.Called(ctx, input, opts)
	return args.Get(0).(*cognitoidentityprovider.InitiateAuthOutput), args.Error(1)
}

func TestCognitoChecker_Check_Success(t *testing.T) {
	mockCognito := new(MockCognitoClient)

	mockCognito.On("InitiateAuthWithContext", mock.Anything, mock.Anything, mock.Anything).Return(&cognitoidentityprovider.InitiateAuthOutput{}, nil)

	checker := CognitoChecker{
		Timeout:          1 * time.Second,
		CheckerName:      "test-cognito-check",
		CognitoAPI:       mockCognito,
		UserPoolId:       "pool-id",
		CognitoClientId:  "client-id",
		TestUserName:     "test-user",
		TestUserPassword: "test-password",
	}

	err := checker.Check(context.Background())
	assert.NoError(t, err)

	mockCognito.AssertExpectations(t)
	input := mockCognito.Calls[0].Arguments[1].(*cognitoidentityprovider.InitiateAuthInput)
	assert.Equal(t, aws.String("client-id"), input.ClientId)
	assert.Equal(t, aws.String("USER_PASSWORD_AUTH"), input.AuthFlow)
	assert.Equal(t, aws.String("test-user"), input.AuthParameters["USERNAME"])
	assert.Equal(t, aws.String("test-password"), input.AuthParameters["PASSWORD"])
	assert.Equal(t, aws.String("e49G23WY7cnTKl++KQZh+LvbCLRDZ5Hsj/QkkovIJLI="), input.AuthParameters["SECRET_HASH"])
}

func TestCognitoChecker_Check_OtherAWSError(t *testing.T) {
	mockCognito := new(MockCognitoClient)
	awsErr := awserr.New("SomeAWSError", "some aws error", nil)
	mockCognito.On("InitiateAuthWithContext", mock.Anything, mock.Anything, mock.Anything).Return((*cognitoidentityprovider.InitiateAuthOutput)(nil), awsErr)

	checker := CognitoChecker{CognitoAPI: mockCognito, CheckerName: "test", Timeout: time.Duration(10) * time.Second}
	err := checker.Check(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "InitiateAuthWithContext failed")
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
	mockCognito.On("InitiateAuthWithContext", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			time.Sleep(200 * time.Millisecond)
		}).
		Return((*cognitoidentityprovider.InitiateAuthOutput)(nil), context.DeadlineExceeded)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	checker := CognitoChecker{CognitoAPI: mockCognito, UserPoolId: "test", CheckerName: "Test", Timeout: 5 * time.Second}

	err := checker.Check(ctx)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "InitiateAuthWithContext failed")
	mockCognito.AssertExpectations(t)
}
