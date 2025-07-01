package health

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider/cognitoidentityprovideriface"
)

// CognitoPermissionChecker verifies permissions like ListUsers.
type CognitoPermissionChecker struct {
	CheckerName string
	UserPoolId  string
	CognitoAPI  cognitoidentityprovideriface.CognitoIdentityProviderAPI
}

// Check verifies that the configured AWS credentials have sufficient permissions
// to perform a Cognito ListUsers operation on the specified User Pool.
func (c *CognitoPermissionChecker) Check(ctx context.Context) error {
	input := &cognitoidentityprovider.ListUsersInput{
		UserPoolId: aws.String(c.UserPoolId),
		Limit:      aws.Int64(1),
	}

	_, err := c.CognitoAPI.ListUsersWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("ListUsersWithContext failed: %w", err)
	}
	return nil
}

// Name returns the unique name of the Cognito permission health check.
func (c *CognitoPermissionChecker) Name() string {
	return c.CheckerName
}

// Type returns the type identifier for the health check.
func (c *CognitoPermissionChecker) Type() string {
	return "cognito-permissions"
}
