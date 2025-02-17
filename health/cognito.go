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
	Timeout         time.Duration
	CheckerName     string
	UserPoolId      string
	CognitoClientID string
	CognitoAPI      cognitoidentityprovideriface.CognitoIdentityProviderAPI
}

// Check queries Cognito for the provided user pool and its client.
// Verifies the connection to Cognito, and that the user pool ID and the user pool client are correct.
func (c *CognitoChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	_, err := c.CognitoAPI.DescribeUserPoolClientWithContext(ctx, &cognitoidentityprovider.DescribeUserPoolClientInput{
		UserPoolId: aws.String(c.UserPoolId),
		ClientId:   aws.String(c.CognitoClientID),
	})

	if err != nil {
		err = fmt.Errorf("cognito DescribeUserPoolClientWithContext failed: %w", err)
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
