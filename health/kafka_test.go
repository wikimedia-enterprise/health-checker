package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockProducer mocks ProducerClient
type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
	args := m.Called(topic, allTopics, timeoutMs)
	return args.Get(0).(*kafka.Metadata), args.Error(1)
}

// MockConsumer mocks ConsumerClient
type MockConsumer struct {
	mock.Mock
}

func (m *MockConsumer) Assignment() ([]kafka.TopicPartition, error) {
	args := m.Called()
	return args.Get(0).([]kafka.TopicPartition), args.Error(1)
}

func (m *MockConsumer) Position(partitions []kafka.TopicPartition) ([]kafka.TopicPartition, error) {
	args := m.Called(partitions)
	return args.Get(0).([]kafka.TopicPartition), args.Error(1)
}

func (m *MockConsumer) Committed(partitions []kafka.TopicPartition, timeoutMs int) ([]kafka.TopicPartition, error) {
	args := m.Called(partitions, timeoutMs)
	return args.Get(0).([]kafka.TopicPartition), args.Error(1)
}

// KafkaCheckerTestSuite tests Kafka health checker
type KafkaCheckerTestSuite struct {
	suite.Suite
	mockProducer *MockProducer
	mockConsumer *MockConsumer
	checker      *KafkaChecker
}

func (s *KafkaCheckerTestSuite) SetupTest() {
	s.mockProducer = new(MockProducer)
	s.mockConsumer = new(MockConsumer)

	cfg := SyncKafkaChecker{
		Name:           "test-kafka",
		Interval:       60 * time.Second,
		Timeout:        20 * time.Second,
		Producer:       s.mockProducer,
		Consumer:       s.mockConsumer,
		RequiredTopics: []string{"topic1"},
		MaxLag:         10,
	}

	s.checker = NewSyncKafkaChecker(cfg, NewConsumerOffsetStore())
}

func (s *KafkaCheckerTestSuite) TestKafkaChecker_Name() {
	s.Equal("test-kafka", s.checker.Name())
}

func (s *KafkaCheckerTestSuite) TestKafkaChecker_Type() {
	s.Equal("kafka", s.checker.Type())
}

func (s *KafkaCheckerTestSuite) TestKafkaChecker_Check_ProducerFailure() {
	s.mockProducer.On("GetMetadata", (*string)(nil), true, 5000).Return((*kafka.Metadata)(nil), errors.New("metadata error"))
	err := s.checker.Check(context.Background())
	s.Error(err)
}

func (s *KafkaCheckerTestSuite) TestKafkaChecker_Check_Success() {
	mockMetadata := &kafka.Metadata{Topics: map[string]kafka.TopicMetadata{"topic1": {}}}
	s.mockProducer.On("GetMetadata", (*string)(nil), true, 5000).Return(mockMetadata, nil)
	mockAssignments := []kafka.TopicPartition{{Topic: stringPtr("topic1"), Partition: 0}}
	s.mockConsumer.On("Assignment").Return(mockAssignments, nil)
	s.mockConsumer.On("Position", mock.Anything).Return(mockAssignments, nil)
	s.mockConsumer.On("Committed", mock.Anything, 5000).Return(mockAssignments, nil)
	err := s.checker.Check(context.Background())
	s.NoError(err)
}

func (s *KafkaCheckerTestSuite) TestKafkaChecker_Check_ConsumerStuck() {
	offsetStore := NewConsumerOffsetStore()
	offsetStore.SetOffsets("topic1", 0, 100, 100)
	mockAssignments := []kafka.TopicPartition{{Topic: stringPtr("topic1"), Partition: 0}}
	currentPosition := kafka.TopicPartition{Topic: stringPtr("topic1"), Partition: 0, Offset: 100}
	committedOffset := kafka.TopicPartition{Topic: stringPtr("topic1"), Partition: 0, Offset: 99}

	s.mockConsumer.On("Position", mockAssignments).Return([]kafka.TopicPartition{currentPosition}, nil)
	s.mockConsumer.On("Committed", mockAssignments, 5000).Return([]kafka.TopicPartition{committedOffset}, nil)
	s.mockConsumer.On("Assignment").Return(mockAssignments, nil)

	err := checkConsumer(s.mockConsumer, offsetStore)
	s.Error(err)
	s.Contains(err.Error(), "consumer appears stuck for partition")
}

func (s *KafkaCheckerTestSuite) TestKafkaChecker_Shutdown() {
	mockMetadata := &kafka.Metadata{Topics: map[string]kafka.TopicMetadata{"topic1": {}, "topic2": {}}}
	s.mockProducer.On("GetMetadata", (*string)(nil), true, 5000).Return(mockMetadata, nil)
	s.mockConsumer.On("Assignment").Return([]kafka.TopicPartition{}, errors.New("assignment error"))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	go func() { _ = s.checker.Check(ctx) }()
	time.Sleep(500 * time.Millisecond)
	s.checker.Shutdown()
}

func stringPtr(s string) *string {
	return &s
}

func TestKafkaCheckerTestSuite(t *testing.T) {
	suite.Run(t, new(KafkaCheckerTestSuite))
}

// AsyncKafkaCheckerTestSuite tests async Kafka health checker
type AsyncKafkaCheckerTestSuite struct {
	suite.Suite
	checker *AsyncKafkaChecker
}

func (suite *AsyncKafkaCheckerTestSuite) SetupTest() {
	suite.checker = NewAsyncKafkaChecker(&KafkaChecker{})
}

func (suite *AsyncKafkaCheckerTestSuite) TestAsyncKafkaCheckerStart() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	suite.checker.Start(ctx)
	time.Sleep(50 * time.Millisecond)
}

func (suite *AsyncKafkaCheckerTestSuite) TestAsyncKafkaCheckerMethods() {
	suite.Equal("kafka-async", suite.checker.Type())
	suite.NotNil(suite.checker.Name())
	suite.NoError(suite.checker.Check(context.Background()))
}

func TestAsyncKafkaCheckerTestSuite(t *testing.T) {
	suite.Run(t, new(AsyncKafkaCheckerTestSuite))
}
