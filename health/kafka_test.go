package health

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"

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

func (m *MockConsumer) QueryWatermarkOffsets(topic string, partition int32, timeoutMs int) (low, high int64, err error) {
	args := m.Called(topic)
	return 0, 0, args.Error(0)
}

// KafkaCheckerTestSuite tests Kafka health checker
type KafkaCheckerTestSuite struct {
	suite.Suite
	mockProducer *MockProducer
	mockConsumer *MockConsumer
	cfg          SyncKafkaChecker
	checker      *KafkaChecker
}

func (s *KafkaCheckerTestSuite) SetupTest() {
	s.mockProducer = new(MockProducer)
	s.mockConsumer = new(MockConsumer)

	s.cfg = SyncKafkaChecker{
		Name:           "test-kafka",
		Interval:       60 * time.Second,
		Producer:       s.mockProducer,
		Consumer:       s.mockConsumer,
		ConsumerTopics: []string{"consumed-topic"},
		ProducerTopics: []string{"topic1"},
		MaxLag:         10,
	}

	var err error
	s.checker, err = NewSyncKafkaChecker(s.cfg, NewConsumerOffsetStore())
	s.NoError(err)
}

func (s *KafkaCheckerTestSuite) TestKafkaChecker_SettingsValidation() {
	cfg := SyncKafkaChecker{
		Name:                  "test-kafka",
		Interval:              60 * time.Second,
		Producer:              s.mockProducer,
		Consumer:              s.mockConsumer,
		ProducerTopics:        []string{"topic1"},
		ConsumerTopics:        []string{"consumed-topic"},
		MaxLag:                10,
		ConsumerIgnoreOffsets: true,
	}
	_, err := NewSyncKafkaChecker(cfg, NewConsumerOffsetStore())
	s.Error(err)
	s.Contains(err.Error(), "incompatible kafka checker consumer settings")

	cfg = SyncKafkaChecker{
		Name:                  "test-kafka",
		Interval:              60 * time.Second,
		Producer:              s.mockProducer,
		Consumer:              s.mockConsumer,
		ProducerTopics:        []string{"topic1"},
		ConsumerTopics:        []string{"consumed-topic"},
		ConsumerIgnoreOffsets: false,
	}
	_, err = NewSyncKafkaChecker(cfg, NewConsumerOffsetStore())
	s.Error(err)
	s.Contains(err.Error(), "incompatible kafka checker consumer settings")

	cfg = SyncKafkaChecker{
		Name:           "test-kafka",
		Interval:       60 * time.Second,
		Producer:       s.mockProducer,
		Consumer:       s.mockConsumer,
		ProducerTopics: []string{"topic1"},
		ConsumerTopics: []string{},
		MaxLag:         5,
	}
	_, err = NewSyncKafkaChecker(cfg, NewConsumerOffsetStore())
	s.Error(err)
	s.Contains(err.Error(), "kafka consumer client without consumer topics to check")

	cfg = SyncKafkaChecker{
		Name:           "test-kafka",
		Interval:       60 * time.Second,
		Producer:       s.mockProducer,
		Consumer:       nil,
		ProducerTopics: []string{"topic1"},
		ConsumerTopics: []string{"consumed-topic"},
		MaxLag:         5,
	}
	_, err = NewSyncKafkaChecker(cfg, NewConsumerOffsetStore())
	s.Error(err)
	s.Contains(err.Error(), "consumer settings provided without consumer client")

	cfg = SyncKafkaChecker{
		Name:           "test-kafka",
		Interval:       60 * time.Second,
		Producer:       s.mockProducer,
		ProducerTopics: []string{},
	}
	_, err = NewSyncKafkaChecker(cfg, NewConsumerOffsetStore())
	s.Error(err)
	s.Contains(err.Error(), "kafka producer client without producer topics to check")

	cfg = SyncKafkaChecker{
		Name:           "test-kafka",
		Interval:       60 * time.Second,
		Producer:       nil,
		ProducerTopics: []string{"topic1"},
	}
	_, err = NewSyncKafkaChecker(cfg, NewConsumerOffsetStore())
	s.Error(err)
	s.Contains(err.Error(), "producer topics to check without a kafka producer client")
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
	mockAssignments := []kafka.TopicPartition{{Topic: stringPtr("consumed-topic"), Partition: 0}}
	s.mockConsumer.On("Assignment").Return(mockAssignments, nil)
	s.mockConsumer.On("Position", mock.Anything).Return(mockAssignments, nil)
	s.mockConsumer.On("Committed", mock.Anything, 5000).Return(mockAssignments, nil)
	err := s.checker.Check(context.Background())
	s.NoError(err)
}

func (s *KafkaCheckerTestSuite) TestKafkaChecker_Check_ConsumerStuck() {
	offsetStore := NewConsumerOffsetStore()
	offsetStore.SetOffsets("consumed-topic", 0, 100, 100)
	var err error
	s.cfg.Producer = nil
	s.cfg.ProducerTopics = nil
	s.checker, err = NewSyncKafkaChecker(s.cfg, offsetStore)
	s.Nil(err)
	mockAssignments := []kafka.TopicPartition{{Topic: stringPtr("consumed-topic"), Partition: 0}}
	currentPosition := kafka.TopicPartition{Topic: stringPtr("consumed-topic"), Partition: 0, Offset: 100}
	committedOffset := kafka.TopicPartition{Topic: stringPtr("consumed-topic"), Partition: 0, Offset: 99}

	s.mockConsumer.On("Position", mockAssignments).Return([]kafka.TopicPartition{currentPosition}, nil)
	s.mockConsumer.On("Committed", mockAssignments, 5000).Return([]kafka.TopicPartition{committedOffset}, nil)
	s.mockConsumer.On("Assignment").Return(mockAssignments, nil)

	err = s.checker.Check(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "consumer appears stuck for partition")
}

func (s *KafkaCheckerTestSuite) TestKafkaChecker_Check_ConsumerStuck_NoOffsetCheck() {
	offsetStore := NewConsumerOffsetStore()
	offsetStore.SetOffsets("consumed-topic", 0, 100, 100)
	var err error
	s.cfg.Producer = nil
	s.cfg.ProducerTopics = nil
	s.cfg.ConsumerIgnoreOffsets = true
	s.cfg.MaxLag = 0
	s.checker, err = NewSyncKafkaChecker(s.cfg, offsetStore)
	s.Nil(err)
	mockAssignments := []kafka.TopicPartition{{Topic: stringPtr("consumed-topic"), Partition: 0}}
	currentPosition := kafka.TopicPartition{Topic: stringPtr("consumed-topic"), Partition: 0, Offset: 100}
	committedOffset := kafka.TopicPartition{Topic: stringPtr("consumed-topic"), Partition: 0, Offset: 99}

	s.mockConsumer.On("Position", mockAssignments).Return([]kafka.TopicPartition{currentPosition}, nil)
	s.mockConsumer.On("Committed", mockAssignments, 5000).Return([]kafka.TopicPartition{committedOffset}, nil)
	s.mockConsumer.On("Assignment").Return(mockAssignments, nil)

	err = s.checker.Check(context.Background())
	s.Nil(err)
}

func (s *KafkaCheckerTestSuite) TestKafkaChecker_Check_NoAssignmentsForTopic() {
	offsetStore := NewConsumerOffsetStore()
	offsetStore.SetOffsets("consumed-topic", 0, 100, 100)
	var err error
	s.cfg.Producer = nil
	s.cfg.ProducerTopics = nil
	s.checker, err = NewSyncKafkaChecker(s.cfg, offsetStore)
	s.Nil(err)
	mockAssignments := []kafka.TopicPartition{{Topic: stringPtr("consumed-topic-2"), Partition: 0}}
	s.mockConsumer.On("Assignment").Return(mockAssignments, nil)

	err = s.checker.Check(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "no assignments for topic consumed-topic")
}

func (s *KafkaCheckerTestSuite) TestKafkaChecker_Check_NoAssignmentsForTopic_NoOffsetCheck() {
	offsetStore := NewConsumerOffsetStore()
	offsetStore.SetOffsets("consumed-topic", 0, 100, 100)
	var err error
	s.cfg.Producer = nil
	s.cfg.ProducerTopics = nil
	s.cfg.ConsumerIgnoreOffsets = true
	s.cfg.MaxLag = 0
	s.checker, err = NewSyncKafkaChecker(s.cfg, offsetStore)
	s.Nil(err)
	mockAssignments := []kafka.TopicPartition{{Topic: stringPtr("consumed-topic-2"), Partition: 0}}
	s.mockConsumer.On("Assignment").Return(mockAssignments, nil)

	err = s.checker.Check(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "no assignments for topic consumed-topic")
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

type AccessCheckTestSuite struct {
	suite.Suite
	mockConsumer *MockConsumer
	checker      *KafkaChecker

	topics            []string
	queryWmOffsetsErr error
	expectedErr       error
}

func (s *AccessCheckTestSuite) SetupTest() {
	s.mockConsumer = new(MockConsumer)

	cfg := SyncKafkaChecker{
		Name:                    "test-kafka",
		Interval:                60 * time.Second,
		Producer:                nil,
		Consumer:                s.mockConsumer,
		ConsumerTopics:          s.topics,
		ConsumerIgnoreOffsets:   true,
		ConsumerCheckReadAccess: true,
	}

	var err error
	s.checker, err = NewSyncKafkaChecker(cfg, NewConsumerOffsetStore())
	s.NoError(err)

	assignments := []kafka.TopicPartition{}
	for _, tpc := range s.topics {
		assignments = append(assignments, kafka.TopicPartition{Topic: &tpc, Partition: 0})
	}
	s.mockConsumer.On("Assignment").Return(assignments, nil)
}

func (s *AccessCheckTestSuite) TearDownTest() {
	s.mockConsumer.AssertExpectations(s.T())
}

func (s *AccessCheckTestSuite) TestKafkaConsumerAccessCheck() {
	for _, tpc := range s.topics {
		s.mockConsumer.On("QueryWatermarkOffsets", tpc).Return(s.queryWmOffsetsErr)
	}

	err := s.checker.Check(context.Background())
	if s.expectedErr == nil {
		s.NoError(err)
	} else {
		s.Error(err, s.expectedErr)
	}
}

func TestKafkaConsumerAccessCheck(t *testing.T) {
	for _, testcase := range []*AccessCheckTestSuite{
		{
			topics: []string{"topic"},
		},
		{
			topics:            []string{"topic"},
			queryWmOffsetsErr: fmt.Errorf("error from Kafka"),
			expectedErr:       fmt.Errorf("consumer check failed: error from Kafka"),
		},
	} {
		suite.Run(t, testcase)
	}
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

// KafkaCheckerSetupTestSuite tests SetUpKafkaCheckers function
type KafkaCheckerSetupTestSuite struct {
	suite.Suite
	mockProducer   *MockProducer
	mockConsumer   *MockConsumer
	consumerTopics []string
	producerTopics []string
}

func (s *KafkaCheckerSetupTestSuite) SetupTest() {

	s.mockProducer = new(MockProducer)
	s.mockConsumer = new(MockConsumer)
	s.consumerTopics = []string{"test-topic-3"}
	s.producerTopics = []string{"test-topic-1", "test-topic-2"}
}

func (s *KafkaCheckerSetupTestSuite) TestSetUpKafkaCheckers() {
	ctx := context.Background()
	intervalMs := 2000
	lag := 5

	mockMetadata := &kafka.Metadata{Topics: map[string]kafka.TopicMetadata{"test-topic-1": {}, "test-topic-2": {}}}
	s.mockProducer.On("GetMetadata", (*string)(nil), true, 5000).Return(mockMetadata, nil)
	mockAssignments := []kafka.TopicPartition{{Topic: &s.consumerTopics[0], Partition: 0}}
	s.mockConsumer.On("Assignment").Return(mockAssignments, nil)
	s.mockConsumer.On("Position", mock.Anything).Return(mockAssignments, nil)
	s.mockConsumer.On("Committed", mock.Anything, 5000).Return(mockAssignments, nil)

	async, err := SetUpKafkaCheckers(ctx, s.producerTopics, s.mockProducer, intervalMs, lag, s.consumerTopics, s.mockConsumer, false)
	s.Nil(err)

	s.Require().NotNil(async, "SetUpKafkaCheckers returned nil")
	s.Equal("kafka-health-check", async.Name())
	s.Equal("kafka-async", async.Type())
	s.Equal(time.Duration(intervalMs)*time.Millisecond, async.checker.checker.Interval)
	s.Equal(int64(lag), async.checker.checker.MaxLag)
	s.Equal(s.producerTopics, async.checker.checker.ProducerTopics)
	s.Require().NotNil(async.checker.checker.Producer)
	s.Require().NotNil(async.checker.checker.Consumer)

	async.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	err = async.Check(ctx)
	s.NoError(err)

	s.mockProducer.AssertExpectations(s.T())
	s.mockConsumer.AssertExpectations(s.T())

}

func TestKafkaCheckerSetupTestSuite(t *testing.T) {
	suite.Run(t, new(KafkaCheckerSetupTestSuite))
}
