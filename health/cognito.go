package health

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider/cognitoidentityprovideriface"
)

// CognitoChecker implements the HealthChecker interface for AWS Cognito.
type CognitoChecker struct {
	Timeout     time.Duration
	CheckerName string
	UserPoolId  string
	CognitoAPI  cognitoidentityprovideriface.CognitoIdentityProviderAPI
}

// Check queries Cognito for the provided user pool.
// Verifies the connection to Cognito, and that the user pool ID is correct.
func (c *CognitoChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	_, err := c.CognitoAPI.DescribeUserPoolWithContext(ctx, &cognitoidentityprovider.DescribeUserPoolInput{
		UserPoolId: aws.String(c.UserPoolId),
	})

	if err != nil {
		err = fmt.Errorf("cognito DescribeUserPoolWithContext failed: %w", err)
	}
	return err
}

// Name returns the name of the health check.
func (c *CognitoChecker) Name() string {
	return c.CheckerName
}

// Type returns the type of the health check (cognito).
func (c *CognitoChecker) Type() string {
	return "cognito"
}

// GetTimeOut returns the deadline for the health check.
func (c *CognitoChecker) GetTimeOut() time.Duration {
	return c.Timeout
}
