package health

import (
	"context"
	"errors"
	"log"
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
	if err != nil {
		log.Fatal(err)
	}
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

type KafkaCheckerSetupTestSuite struct {
	suite.Suite
	producer       *kafka.Producer
	consumer       *kafka.Consumer
	requiredTopics []string
}

func (s *KafkaCheckerSetupTestSuite) SetupSuite() {
	producerConfig := &kafka.ConfigMap{
		"bootstrap.servers": getKafkaBootstrapServers(),
	}
	var err error

	s.producer, err = kafka.NewProducer(producerConfig)
	s.Require().NoError(err, "Failed to create Kafka producer")

	consumerConfig := &kafka.ConfigMap{
		"bootstrap.servers":        getKafkaBootstrapServers(),
		"group.id":                 "kafka-checker-setup-test-group",
		"auto.offset.reset":        "earliest",
		"go.events.channel.enable": true,
	}

	s.consumer, err = kafka.NewConsumer(consumerConfig)
	s.Require().NoError(err, "Failed to create Kafka consumer")

	s.requiredTopics = []string{"test-topic-1", "test-topic-2"}
	err = s.consumer.SubscribeTopics(s.requiredTopics, nil)
	s.Require().NoError(err, "Failed to subscribe to topics")

}

func (s *KafkaCheckerSetupTestSuite) TearDownSuite() {
	if s.producer != nil {
		s.producer.Close()
	}
	if s.consumer != nil {
		s.consumer.Close()
	}
}

func (s *KafkaCheckerSetupTestSuite) TestSetUpKafkaCheckers() {
	ctx := context.Background()
	intervalMS := 2000
	lag := 5

	checker := SetUpKafkaCheckers(ctx, s.requiredTopics, s.producer, intervalMS, lag, s.consumer)

	s.Require().NotNil(checker, "SetUpKafkaCheckers returned nil")
	s.Equal("kafka-health-check", checker.Name())
	s.Equal("kafka-async", checker.Type())
	s.Equal(time.Duration(intervalMS)*time.Millisecond, checker.checker.checker.Interval)
	s.Equal(int64(lag), checker.checker.checker.MaxLag)
	s.Equal(s.requiredTopics, checker.checker.checker.RequiredTopics)
	s.Require().NotNil(checker.checker.checker.Producer)
	s.Require().NotNil(checker.checker.checker.Consumer)

	producer, ok := checker.checker.checker.Producer.(*kafka.Producer)
	s.True(ok)
	s.Equal(s.producer, producer)

	consumer, ok := checker.checker.checker.Consumer.(*kafka.Consumer)
	s.True(ok)
	s.Equal(s.consumer, consumer)

	checker.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	err := checker.Check(ctx)
	s.NoError(err)

}

func TestKafkaCheckerSetupTestSuite(t *testing.T) {
	suite.Run(t, new(KafkaCheckerSetupTestSuite))
}

func getKafkaBootstrapServers() string {
	servers := "localhost:9092"
	return servers
}
