package health

import (
	"context"
	"log"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wikimedia-enterprise/health-checker/health/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type fooServer struct {
	protos.UnimplementedTestServiceServer
}

func (f *fooServer) Foo(_ context.Context, req *protos.FooRequest) (*protos.FooResponse, error) {
	if req.GetBar() == "baz" {
		return &protos.FooResponse{}, nil
	}

	return nil, status.Errorf(codes.InvalidArgument, "invalid request: %s", req)
}

func StartGrpcServer(ctx context.Context) (client protos.TestServiceClient, closer func()) {
	buffer := 101024 * 1024
	lis := bufconn.Listen(buffer)

	baseServer := grpc.NewServer()
	protos.RegisterTestServiceServer(baseServer, &fooServer{})

	go func() {
		if err := baseServer.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("error connecting to server: %v", err)
	}

	closer = func() {
		err := lis.Close()
		if err != nil {
			log.Printf("error closing listener: %v", err)
		}
		baseServer.Stop()
	}

	client = protos.NewTestServiceClient(conn)

	return client, closer
}

func TestGrpcCheckSuccess_successResponse(t *testing.T) {
	ctx := context.Background()
	client, closer := StartGrpcServer(ctx)
	defer closer()

	grpcChecker, err := NewGrpcChecker("test", client.Foo, &protos.FooRequest{Bar: "baz"}, []codes.Code{codes.OK})
	assert.Nil(t, err)

	err = grpcChecker.Check(ctx)
	assert.Nil(t, err)
}

func TestGrpcCheckSuccess_successResponse_defaultStatus(t *testing.T) {
	ctx := context.Background()
	client, closer := StartGrpcServer(ctx)
	defer closer()

	// Will default to OK.
	grpcChecker, err := NewGrpcChecker("test", client.Foo, &protos.FooRequest{Bar: "baz"}, nil)
	assert.Nil(t, err)

	err = grpcChecker.Check(ctx)
	assert.Nil(t, err)
}

func TestGrpcCheckSuccess_errorResponse(t *testing.T) {
	ctx := context.Background()
	client, closer := StartGrpcServer(ctx)
	defer closer()

	grpcChecker, err := NewGrpcChecker("test", client.Foo, &protos.FooRequest{}, []codes.Code{codes.InvalidArgument})
	assert.Nil(t, err)

	err = grpcChecker.Check(ctx)
	assert.Nil(t, err)
}

func TestGrpcCheckFails_successResponse(t *testing.T) {
	ctx := context.Background()
	client, closer := StartGrpcServer(ctx)
	defer closer()

	grpcChecker, err := NewGrpcChecker("test", client.Foo, &protos.FooRequest{Bar: "baz"}, []codes.Code{codes.InvalidArgument})
	assert.Nil(t, err)

	err = grpcChecker.Check(ctx)
	assert.Error(t, err)
}

func TestGrpcCheckFails_errorResponse(t *testing.T) {
	ctx := context.Background()
	client, closer := StartGrpcServer(ctx)
	defer closer()

	grpcChecker, err := NewGrpcChecker("test", client.Foo, &protos.FooRequest{}, []codes.Code{codes.NotFound})
	assert.Nil(t, err)

	err = grpcChecker.Check(ctx)
	assert.Error(t, err)
}
