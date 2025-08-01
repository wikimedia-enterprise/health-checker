package health

import (
	"context"
	"errors"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func TestNewRedisChecker_Success(t *testing.T) {
	mockClient, _ := redismock.NewClientMock()

	config := RedisCheckerConfig{Name: "redis-test"}
	checker, err := NewRedisChecker(mockClient, config)

	assert.NoError(t, err)
	assert.NotNil(t, checker)
	assert.Equal(t, "redis-test", checker.Name())

}

func TestNewRedisChecker_Failure(t *testing.T) {
	config := RedisCheckerConfig{Name: "redis-test"}
	checker, err := NewRedisChecker(nil, config)

	assert.Error(t, err)
	assert.Nil(t, checker)
}

func TestCheck_Success(t *testing.T) {
	mockClient, mock := redismock.NewClientMock()
	mock.ExpectPing().SetVal("PONG")

	config := RedisCheckerConfig{Name: "redis-test"}
	checker := &RedisChecker{config: config, client: mockClient}

	ctx := context.Background()
	err := checker.Check(ctx)

	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheck_Failure(t *testing.T) {
	mockClient, mock := redismock.NewClientMock()
	mock.ExpectPing().SetErr(errors.New("redis is down"))

	config := RedisCheckerConfig{Name: "redis-test"}
	checker := &RedisChecker{config: config, client: mockClient}

	ctx := context.Background()
	err := checker.Check(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis is down")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestName(t *testing.T) {
	checker := &RedisChecker{config: RedisCheckerConfig{Name: "test-redis"}}
	assert.Equal(t, "test-redis", checker.Name())
}

func TestType(t *testing.T) {
	checker := &RedisChecker{}
	assert.Equal(t, "redis", checker.Type())
}
