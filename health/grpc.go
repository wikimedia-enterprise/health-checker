package health

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewGrpcChecker creates a GrpcChecker with the provided configuration.
func NewGrpcChecker[Req any, Res fmt.Stringer](checkerName string, stub GrpcStub[Req, Res], request Req, expectedStatus codes.Code) (*GrpcChecker[Req, Res], error) {
	return &GrpcChecker[Req, Res]{
		checkerName:    checkerName,
		stub:           stub,
		request:        request,
		expectedStatus: expectedStatus,
	}, nil
}

// GrpcStub matches the signature of a gRPC call.
type GrpcStub[Req any, Res fmt.Stringer] func(ctx context.Context, in Req, opts ...grpc.CallOption) (Res, error)

// GrpcChecker contains the configuration of the health checker.
type GrpcChecker[Req any, Res fmt.Stringer] struct {
	checkerName    string
	stub           GrpcStub[Req, Res]
	request        Req
	expectedStatus codes.Code
}

// Check runs the health check and returns the result.
func (c *GrpcChecker[Req, Res]) Check(ctx context.Context) error {
	res, err := c.stub(ctx, c.request)

	if status.Code(err) == c.expectedStatus {
		return nil
	}

	errorStr := "<nil>"
	if err != nil {
		errorStr = err.Error()
	}
	return fmt.Errorf("expected code %s but got %s instead, full response: '%s', full status '%s'",
		c.expectedStatus, status.Code(err), res, errorStr)
}

// Name returns the name of the health check.
func (c *GrpcChecker[Req, Res]) Name() string {
	return c.checkerName
}

// Type returns the type of the health check (grpc).
func (c *GrpcChecker[Req, Res]) Type() string {
	return "grpc"
}
