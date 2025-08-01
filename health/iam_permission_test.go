package health

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockIAMAPI is a mock implementation of the iamiface.IAMAPI interface
type MockIAMAPI struct {
	iamiface.IAMAPI
	mock.Mock
}

func (m *MockIAMAPI) SimulatePrincipalPolicyWithContext(ctx aws.Context, input *iam.SimulatePrincipalPolicyInput, opts ...request.Option) (*iam.SimulatePolicyResponse, error) {
	args := m.Called(ctx, input, opts)
	return args.Get(0).(*iam.SimulatePolicyResponse), args.Error(1)
}

// MockSTSAPI is a mock implementation of the stsiface.STSAPI interface
type MockSTSAPI struct {
	stsiface.STSAPI
	mock.Mock
}

func (m *MockSTSAPI) GetCallerIdentityWithContext(ctx aws.Context, input *sts.GetCallerIdentityInput, opts ...request.Option) (*sts.GetCallerIdentityOutput, error) {
	args := m.Called(ctx, input, opts)
	return args.Get(0).(*sts.GetCallerIdentityOutput), args.Error(1)
}

func TestIAMPermissionChecker_Check_Success_WithProvidedArn(t *testing.T) {
	mockIAM := new(MockIAMAPI)

	// Mock successful policy simulation
	mockOutput := &iam.SimulatePolicyResponse{
		EvaluationResults: []*iam.EvaluationResult{
			{
				EvalActionName: aws.String("cognito-idp:ListUsers"),
				EvalDecision:   aws.String(iam.PolicyEvaluationDecisionTypeAllowed),
			},
			{
				EvalActionName: aws.String("cognito-idp:GetUser"),
				EvalDecision:   aws.String(iam.PolicyEvaluationDecisionTypeAllowed),
			},
		},
	}

	mockIAM.On("SimulatePrincipalPolicyWithContext", mock.Anything, mock.MatchedBy(func(input *iam.SimulatePrincipalPolicyInput) bool {
		return *input.PolicySourceArn == "arn:aws:iam::123456789012:role/test-role" &&
			len(input.ActionNames) == 2 &&
			*input.ActionNames[0] == "cognito-idp:ListUsers" &&
			*input.ActionNames[1] == "cognito-idp:GetUser"
	}), mock.Anything).Return(mockOutput, nil)

	checker := &IAMPermissionChecker{
		CheckerName:     "test-iam-permission-check",
		PolicySourceArn: "arn:aws:iam::123456789012:role/test-role",
		Actions:         []string{"cognito-idp:ListUsers", "cognito-idp:GetUser"},
		ResourceArns:    []string{"arn:aws:cognito-idp:us-west-2:123456789012:userpool/us-west-2_123456"},
		IAMAPI:          mockIAM,
	}

	err := checker.Check(context.Background())
	assert.NoError(t, err)
	mockIAM.AssertExpectations(t)
}

func TestIAMPermissionChecker_Check_Success_WithCallerIdentity(t *testing.T) {
	mockIAM := new(MockIAMAPI)
	mockSTS := new(MockSTSAPI)

	mockSTSOutput := &sts.GetCallerIdentityOutput{
		Arn: aws.String("arn:aws:iam::123456789012:role/discovered-role"),
	}
	mockSTS.On("GetCallerIdentityWithContext", mock.Anything, mock.Anything, mock.Anything).Return(mockSTSOutput, nil)

	mockOutput := &iam.SimulatePolicyResponse{
		EvaluationResults: []*iam.EvaluationResult{
			{
				EvalActionName: aws.String("s3:GetObject"),
				EvalDecision:   aws.String(iam.PolicyEvaluationDecisionTypeAllowed),
			},
		},
	}

	mockIAM.On("SimulatePrincipalPolicyWithContext", mock.Anything, mock.MatchedBy(func(input *iam.SimulatePrincipalPolicyInput) bool {
		return *input.PolicySourceArn == "arn:aws:iam::123456789012:role/discovered-role"
	}), mock.Anything).Return(mockOutput, nil)

	checker := &IAMPermissionChecker{
		CheckerName:  "test-iam-permission-check",
		Actions:      []string{"s3:GetObject"},
		ResourceArns: []string{"arn:aws:s3:::test-bucket/*"},
		IAMAPI:       mockIAM,
		STSAPI:       mockSTS,
	}

	err := checker.Check(context.Background())
	assert.NoError(t, err)
	mockIAM.AssertExpectations(t)
	mockSTS.AssertExpectations(t)
}

func TestIAMPermissionChecker_Check_Failure_DeniedActions(t *testing.T) {
	mockIAM := new(MockIAMAPI)

	// Mock policy simulation with some denied actions
	mockOutput := &iam.SimulatePolicyResponse{
		EvaluationResults: []*iam.EvaluationResult{
			{
				EvalActionName: aws.String("cognito-idp:ListUsers"),
				EvalDecision:   aws.String(iam.PolicyEvaluationDecisionTypeAllowed),
			},
			{
				EvalActionName: aws.String("cognito-idp:DeleteUser"),
				EvalDecision:   aws.String(iam.PolicyEvaluationDecisionTypeExplicitDeny),
			},
			{
				EvalActionName: aws.String("cognito-idp:AdminDeleteUser"),
				EvalDecision:   aws.String(iam.PolicyEvaluationDecisionTypeImplicitDeny),
			},
		},
	}

	mockIAM.On("SimulatePrincipalPolicyWithContext", mock.Anything, mock.Anything, mock.Anything).Return(mockOutput, nil)

	checker := &IAMPermissionChecker{
		CheckerName:     "test-iam-permission-check",
		PolicySourceArn: "arn:aws:iam::123456789012:role/test-role",
		Actions:         []string{"cognito-idp:ListUsers", "cognito-idp:DeleteUser", "cognito-idp:AdminDeleteUser"},
		IAMAPI:          mockIAM,
	}

	err := checker.Check(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IAM permission check failed")
	assert.Contains(t, err.Error(), "cognito-idp:DeleteUser")
	assert.Contains(t, err.Error(), "cognito-idp:AdminDeleteUser")
	mockIAM.AssertExpectations(t)
}

func TestIAMPermissionChecker_Check_Failure_SimulatePrincipalPolicyError(t *testing.T) {
	mockIAM := new(MockIAMAPI)

	mockIAM.On("SimulatePrincipalPolicyWithContext", mock.Anything, mock.Anything, mock.Anything).Return((*iam.SimulatePolicyResponse)(nil), errors.New("IAM API error"))

	checker := &IAMPermissionChecker{
		CheckerName:     "test-iam-permission-check",
		PolicySourceArn: "arn:aws:iam::123456789012:role/test-role",
		Actions:         []string{"cognito-idp:ListUsers"},
		IAMAPI:          mockIAM,
	}

	err := checker.Check(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SimulatePrincipalPolicy failed")
	assert.Contains(t, err.Error(), "IAM API error")
	mockIAM.AssertExpectations(t)
}

func TestIAMPermissionChecker_Check_Failure_GetCallerIdentityError(t *testing.T) {
	mockSTS := new(MockSTSAPI)

	mockSTS.On("GetCallerIdentityWithContext", mock.Anything, mock.Anything, mock.Anything).Return((*sts.GetCallerIdentityOutput)(nil), errors.New("STS API error"))

	checker := &IAMPermissionChecker{
		CheckerName: "test-iam-permission-check",
		Actions:     []string{"s3:GetObject"},
		STSAPI:      mockSTS,
	}

	err := checker.Check(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not get caller identity ARN")
	assert.Contains(t, err.Error(), "STS API error")
	mockSTS.AssertExpectations(t)
}

func TestIAMPermissionChecker_Check_Failure_MissingSTSAPI(t *testing.T) {
	checker := &IAMPermissionChecker{
		CheckerName: "test-iam-permission-check",
		Actions:     []string{"s3:GetObject"},
	}

	err := checker.Check(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "STSAPI client is required to get caller identity when PolicySourceArn is not provided")
}

func TestIAMPermissionChecker_Check_WithResourceArns(t *testing.T) {
	mockIAM := new(MockIAMAPI)

	// Mock successful policy simulation
	mockOutput := &iam.SimulatePolicyResponse{
		EvaluationResults: []*iam.EvaluationResult{
			{
				EvalActionName: aws.String("s3:GetObject"),
				EvalDecision:   aws.String(iam.PolicyEvaluationDecisionTypeAllowed),
			},
		},
	}

	expectedResourceArns := []string{
		"arn:aws:s3:::test-bucket/*",
		"arn:aws:s3:::another-bucket/prefix/*",
	}

	mockIAM.On("SimulatePrincipalPolicyWithContext", mock.Anything, mock.MatchedBy(func(input *iam.SimulatePrincipalPolicyInput) bool {
		if len(input.ResourceArns) != 2 {
			return false
		}
		return *input.ResourceArns[0] == expectedResourceArns[0] &&
			*input.ResourceArns[1] == expectedResourceArns[1]
	}), mock.Anything).Return(mockOutput, nil)

	checker := &IAMPermissionChecker{
		CheckerName:     "test-iam-permission-check",
		PolicySourceArn: "arn:aws:iam::123456789012:role/test-role",
		Actions:         []string{"s3:GetObject"},
		ResourceArns:    expectedResourceArns,
		IAMAPI:          mockIAM,
	}

	err := checker.Check(context.Background())
	assert.NoError(t, err)
	mockIAM.AssertExpectations(t)
}

func TestIAMPermissionChecker_Check_WithoutResourceArns(t *testing.T) {
	mockIAM := new(MockIAMAPI)

	// Mock successful policy simulation
	mockOutput := &iam.SimulatePolicyResponse{
		EvaluationResults: []*iam.EvaluationResult{
			{
				EvalActionName: aws.String("iam:ListRoles"),
				EvalDecision:   aws.String(iam.PolicyEvaluationDecisionTypeAllowed),
			},
		},
	}

	mockIAM.On("SimulatePrincipalPolicyWithContext", mock.Anything, mock.MatchedBy(func(input *iam.SimulatePrincipalPolicyInput) bool {
		return input.ResourceArns == nil
	}), mock.Anything).Return(mockOutput, nil)

	checker := &IAMPermissionChecker{
		CheckerName:     "test-iam-permission-check",
		PolicySourceArn: "arn:aws:iam::123456789012:role/test-role",
		Actions:         []string{"iam:ListRoles"},
		IAMAPI:          mockIAM,
	}

	err := checker.Check(context.Background())
	assert.NoError(t, err)
	mockIAM.AssertExpectations(t)
}

func TestIAMPermissionChecker_NameAndType(t *testing.T) {
	checker := &IAMPermissionChecker{
		CheckerName: "my-iam-checker",
	}

	assert.Equal(t, "my-iam-checker", checker.Name())
	assert.Equal(t, "iam-permissions", checker.Type())
}

func TestGetCallerIdentityArn_Success(t *testing.T) {
	mockSTS := new(MockSTSAPI)

	mockOutput := &sts.GetCallerIdentityOutput{
		Arn: aws.String("arn:aws:iam::123456789012:role/test-role"),
	}

	mockSTS.On("GetCallerIdentityWithContext", mock.Anything, mock.Anything, mock.Anything).Return(mockOutput, nil)

	arn, err := getCallerIdentityArn(context.Background(), mockSTS)
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:iam::123456789012:role/test-role", arn)
	mockSTS.AssertExpectations(t)
}

func TestGetCallerIdentityArn_Failure(t *testing.T) {
	mockSTS := new(MockSTSAPI)

	mockSTS.On("GetCallerIdentityWithContext", mock.Anything, mock.Anything, mock.Anything).Return((*sts.GetCallerIdentityOutput)(nil), errors.New("STS error"))

	arn, err := getCallerIdentityArn(context.Background(), mockSTS)
	assert.Error(t, err)
	assert.Empty(t, arn)
	assert.Contains(t, err.Error(), "STS error")
	mockSTS.AssertExpectations(t)
}

func TestGetCallerIdentityArn_NilArn(t *testing.T) {
	mockSTS := new(MockSTSAPI)

	mockOutput := &sts.GetCallerIdentityOutput{
		Arn: nil,
	}

	mockSTS.On("GetCallerIdentityWithContext", mock.Anything, mock.Anything, mock.Anything).Return(mockOutput, nil)

	arn, err := getCallerIdentityArn(context.Background(), mockSTS)
	assert.NoError(t, err)
	assert.Empty(t, arn)
	mockSTS.AssertExpectations(t)
}
