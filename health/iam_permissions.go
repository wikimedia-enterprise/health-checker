package health

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
)

// IAMPermissionChecker verifies a set of IAM permissions using SimulatePrincipalPolicy.
type IAMPermissionChecker struct {
	CheckerName     string
	PolicySourceArn string   // ARN of the principal (role) whose permissions are being checked. Can be left empty to use the application's current identity.
	Actions         []string // List of IAM actions to check (e.g., "cognito-idp:ListUsers").
	ResourceArns    []string
	IAMAPI          iamiface.IAMAPI
	STSAPI          stsiface.STSAPI
}

// Check verifies that the configured IAM principal has sufficient permissions.
func (c *IAMPermissionChecker) Check(ctx context.Context) error {
	policySourceArn := c.PolicySourceArn
	var err error

	if policySourceArn == "" {
		if c.STSAPI == nil {
			return fmt.Errorf("STSAPI client is required to get caller identity when PolicySourceArn is not provided")
		}
		policySourceArn, err = getCallerIdentityArn(ctx, c.STSAPI)
		if err != nil {
			return fmt.Errorf("could not get caller identity ARN: %w", err)
		}
	}

	input := &iam.SimulatePrincipalPolicyInput{
		PolicySourceArn: aws.String(policySourceArn),
		ActionNames:     aws.StringSlice(c.Actions),
	}

	if len(c.ResourceArns) > 0 {
		input.ResourceArns = aws.StringSlice(c.ResourceArns)
	}

	output, err := c.IAMAPI.SimulatePrincipalPolicyWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("SimulatePrincipalPolicy failed: %w", err)
	}

	var deniedActions []string
	for _, result := range output.EvaluationResults {
		if *result.EvalDecision != iam.PolicyEvaluationDecisionTypeAllowed {
			deniedActions = append(deniedActions, *result.EvalActionName)
		}
	}

	if len(deniedActions) > 0 {
		return fmt.Errorf("IAM permission check failed: the following actions are denied: %s", strings.Join(deniedActions, ", "))
	}

	return nil
}

// Name returns the unique name of the IAM permission health check.
func (c *IAMPermissionChecker) Name() string {
	return c.CheckerName
}

// Type returns the type identifier for the health check.
func (c *IAMPermissionChecker) Type() string {
	return "iam-permissions"
}

// getCallerIdentityArn retrieves the ARN of the currently assumed role.
func getCallerIdentityArn(ctx context.Context, stsapi stsiface.STSAPI) (string, error) {
	input := &sts.GetCallerIdentityInput{}
	result, err := stsapi.GetCallerIdentityWithContext(ctx, input)
	if err != nil {
		return "", err
	}
	return aws.StringValue(result.Arn), nil
}
