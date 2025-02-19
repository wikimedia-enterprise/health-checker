package health

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider/cognitoidentityprovideriface"
)

// CognitoChecker implements the HealthChecker interface for AWS Cognito.
type CognitoChecker struct {
	Timeout          time.Duration
	CheckerName      string
	UserPoolId       string
	CognitoClientId  string
	CognitoSecret    string
	TestUserName     string
	TestUserPassword string
	CognitoAPI       cognitoidentityprovideriface.CognitoIdentityProviderAPI
}

// Check verifies the authentication flow with Cognito for a given username and password.
func (c *CognitoChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	h := hmac.New(sha256.New, []byte(c.CognitoSecret))

	if _, err := h.Write([]byte(fmt.Sprintf("%s%s", c.TestUserName, c.CognitoClientId))); err != nil {
		return fmt.Errorf("error writing to hmac hash %w", err)
	}

	_, err := c.CognitoAPI.InitiateAuthWithContext(ctx, &cognitoidentityprovider.InitiateAuthInput{
		ClientId: aws.String(c.CognitoClientId),
		AuthFlow: aws.String("USER_PASSWORD_AUTH"),
		AuthParameters: map[string]*string{
			"USERNAME":    aws.String(c.TestUserName),
			"PASSWORD":    aws.String(c.TestUserPassword),
			"SECRET_HASH": aws.String(base64.StdEncoding.EncodeToString(h.Sum(nil))),
		},
	})

	if err != nil {
		return fmt.Errorf("InitiateAuthWithContext failed: %w", err)
	}
	return nil
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
