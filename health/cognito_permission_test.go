package health

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider/cognitoidentityprovideriface"
	"github.com/stretchr/testify/assert"
)

// mockCognitoAPI is a mock implementation of the CognitoIdentityProviderAPI interface
type mockCognitoAPI struct {
	cognitoidentityprovideriface.CognitoIdentityProviderAPI
	ListUsersFunc func(*cognitoidentityprovider.ListUsersInput) (*cognitoidentityprovider.ListUsersOutput, error)
}

func (m *mockCognitoAPI) ListUsersWithContext(_ context.Context, input *cognitoidentityprovider.ListUsersInput, _ ...request.Option) (*cognitoidentityprovider.ListUsersOutput, error) {
	return m.ListUsersFunc(input)
}

func TestCognitoPermissionChecker_Check_Success(t *testing.T) {
	mockAPI := &mockCognitoAPI{
		ListUsersFunc: func(input *cognitoidentityprovider.ListUsersInput) (*cognitoidentityprovider.ListUsersOutput, error) {
			// Simulate successful response
			return &cognitoidentityprovider.ListUsersOutput{}, nil
		},
	}

	checker := &CognitoPermissionChecker{
		CheckerName: "test-cognito-permission-check",
		UserPoolId:  "us-west-2_123456",
		CognitoAPI:  mockAPI,
	}

	err := checker.Check(context.Background())
	assert.NoError(t, err)
}

func TestCognitoPermissionChecker_Check_Failure(t *testing.T) {
	mockAPI := &mockCognitoAPI{
		ListUsersFunc: func(input *cognitoidentityprovider.ListUsersInput) (*cognitoidentityprovider.ListUsersOutput, error) {
			// Simulate IAM failure
			return nil, errors.New("access denied")
		},
	}

	checker := &CognitoPermissionChecker{
		CheckerName: "test-cognito-permission-check",
		UserPoolId:  "us-west-2_123456",
		CognitoAPI:  mockAPI,
	}

	err := checker.Check(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestCognitoPermissionChecker_NameAndType(t *testing.T) {
	checker := &CognitoPermissionChecker{
		CheckerName: "my-checker",
	}

	assert.Equal(t, "my-checker", checker.Name())
	assert.Equal(t, "cognito-permissions", checker.Type())
}
