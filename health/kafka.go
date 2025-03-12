package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hellofresh/health-go/v5"
)

type ConsumerClient interface {
	Assignment() ([]kafka.TopicPartition, error)
	Position(partitions []kafka.TopicPartition) (offsets []kafka.TopicPartition, err error)
	Committed(partitions []kafka.TopicPartition, timeoutMs int) (offsets []kafka.TopicPartition, err error)
}

type ProducerClient interface {
	GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error)
}

// SyncKafkaChecker holds the configuration for a KafkaChecker.
type SyncKafkaChecker struct {
	Name     string        // Name of this health check.
	Interval time.Duration // Interval for the health check, only for repeated checks.

	Producer       ProducerClient
	Consumer       ConsumerClient
	RequiredTopics []string
	MaxLag         int64
}

// KafkaChecker holds the configuration for a KafkaChecker with a stop channel.
type KafkaChecker struct {
	checker     SyncKafkaChecker
	offsetStore *ConsumerOffsetStore

	stopChan chan struct{}
	wg       sync.WaitGroup
}

type AsyncKafkaCheckerConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
}

// DefaultAsyncKafkaCheckerConfig provides default values for the asynchronous checker.
var DefaultAsyncKafkaCheckerConfig = AsyncKafkaCheckerConfig{
	MaxRetries:     5,
	InitialBackoff: 1 * time.Second,
}

// RegisterKafkaHealthChecks registers Kafka health checks with the health package.
func RegisterKafkaHealthChecks(h *health.Health, configs []SyncKafkaChecker, isAsync bool, asyncConfig ...AsyncKafkaCheckerConfig) error {
	// Use default config if none provided, otherwise use the first one.
	var config AsyncKafkaCheckerConfig
	if len(asyncConfig) > 0 {
		config = asyncConfig[0]
	} else {
		config = DefaultAsyncKafkaCheckerConfig
	}

	for _, conf := range configs {
		var checker HealthChecker
		syncChecker := NewSyncKafkaChecker(conf, NewConsumerOffsetStore())
		if isAsync {
			// Pass maxRetries and initialBackoff here!
			checker = NewAsyncKafkaChecker(syncChecker, config.MaxRetries, config.InitialBackoff)
		} else {
			checker = syncChecker
		}

		checkConfig := health.Config{
			Name:    checker.Name(),
			Check:   checker.Check,
			Timeout: NoTimeout,
		}

		if err := h.Register(checkConfig); err != nil {
			return fmt.Errorf("failed to register Kafka check %q: %w", checkConfig.Name, err)
		}
	}
	return nil
}

// NewKafkaChecker creates a new KafkaChecker struct
func NewSyncKafkaChecker(checker SyncKafkaChecker, store *ConsumerOffsetStore) *KafkaChecker {
	return &KafkaChecker{
		checker:     checker,
		offsetStore: store,
		stopChan:    make(chan struct{}),
	}
}

// Check performs a Kafka health check.
func (k *KafkaChecker) Check(ctx context.Context) error {
	if k.checker.Producer != nil {
		if err := checkProducer(k.checker.Producer, k.checker.RequiredTopics); err != nil {
			return fmt.Errorf("producer check failed: %w", err)
		}
	}

	if k.checker.Consumer != nil {
		if err := checkConsumer(k.checker.Consumer, k.offsetStore); err != nil {
			return fmt.Errorf("consumer check failed: %w", err)
		}
	}

	return nil
}

// Name returns the name of the Kafka health check.
func (k *KafkaChecker) Name() string { return k.checker.Name }

// Type returns the type of the Kafka health check.
func (k *KafkaChecker) Type() string { return "kafka" }

// Shutdown stops the Kafka health check.
func (k *KafkaChecker) Shutdown() {
	close(k.stopChan)
	k.wg.Wait()
}

// checkProducer checks that the producer can access the required topics.
func checkProducer(producer ProducerClient, requiredTopics []string) error {
	metadata, err := producer.GetMetadata(nil, true, 5000)
	if err != nil {
		return err
	}

	for _, topic := range requiredTopics {
		found := false
		for metadataTopic := range metadata.Topics {
			if metadataTopic == topic {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("required topic %s not found", topic)
		}
	}
	return nil
}

// checkConsumer checks the current and committed offsets for all partitions assigned to a consumer.
func checkConsumer(consumer ConsumerClient, store *ConsumerOffsetStore) error {
	assignments, err := consumer.Assignment()
	if err != nil {
		return err
	}

	if len(assignments) == 0 {
		return fmt.Errorf("consumer has no partition assignments")
	}

	for _, assignment := range assignments {
		if err := checkPartition(consumer, store, assignment); err != nil {
			return err
		}
	}

	return nil
}

// checkPartition checks the current and committed offsets for a partition.
func checkPartition(consumer ConsumerClient, store *ConsumerOffsetStore, assignment kafka.TopicPartition) error {
	position, err := consumer.Position([]kafka.TopicPartition{assignment})
	if err != nil {
		return fmt.Errorf("failed to get position for partition %v: %w", assignment, err)
	}

	committed, err := consumer.Committed([]kafka.TopicPartition{assignment}, 5000)
	if err != nil {
		return fmt.Errorf("failed to get committed offset for partition %v: %w", assignment, err)
	}

	currentPosition := position[0].Offset
	committedOffset := committed[0].Offset

	return evaluateOffsets(store, assignment, currentPosition, committedOffset)
}

// evaluateOffsets compares the current and committed offsets for a partition.
func evaluateOffsets(store *ConsumerOffsetStore, assignment kafka.TopicPartition, position, commit kafka.Offset) error {
	prevPosition, prevCommit, exists := store.GetOffsets(*assignment.Topic, assignment.Partition)

	if !exists || position > prevPosition || commit > prevCommit {
		store.SetOffsets(*assignment.Topic, assignment.Partition, position, commit)
		return nil
	}

	if position == commit {
		return nil
	}

	return fmt.Errorf("consumer appears stuck for partition %v (position: %d, committed: %d)",
		assignment, position, commit)
}

// AsyncKafkaChecker is an asynchronous health checker for Kafka.
type AsyncKafkaChecker struct {
	checker        *KafkaChecker
	store          *AsyncHealthStore
	maxRetries     int
	initialBackoff time.Duration
}

// NewAsyncKafkaChecker creates a new AsyncKafkaChecker.
func NewAsyncKafkaChecker(checker *KafkaChecker, maxRetries int, initialBackoff time.Duration) *AsyncKafkaChecker {
	return &AsyncKafkaChecker{
		checker:        checker,
		store:          NewAsyncHealthStore(),
		maxRetries:     maxRetries,
		initialBackoff: initialBackoff,
	}
}

// Start begins the asynchronous Kafka health check loop.
func (akc *AsyncKafkaChecker) Start(ctx context.Context) {
	var startupErr error
	for i := 0; i < akc.maxRetries; i++ {
		if i > 0 {
			backoff := akc.initialBackoff * time.Duration(1<<uint(i-1))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				akc.store.UpdateStatus(akc.checker.Name(), ctx.Err())
				return
			}
		}

		startupErr = akc.checker.Check(ctx)
		if startupErr == nil {
			break
		}
		if ctx.Err() != nil {
			akc.store.UpdateStatus(akc.checker.Name(), ctx.Err())
			return
		}
	}
	akc.store.UpdateStatus(akc.checker.Name(), startupErr)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(akc.checker.checker.Interval):
				err := akc.checker.Check(ctx)
				akc.store.UpdateStatus(akc.checker.Name(), err)
			}
		}
	}()
}

// Check retrieves the cached health check result.
func (akc *AsyncKafkaChecker) Check(ctx context.Context) error {
	return akc.store.GetStatus(akc.checker.Name())
}

// Name returns the name of the Kafka health check.
func (akc *AsyncKafkaChecker) Name() string {
	return akc.checker.Name()
}

// Type returns the type of the health check.
func (akc *AsyncKafkaChecker) Type() string {
	return "kafka-async"
}

// SetupKafkaCheckers creates an AsyncKafkaChecker
func SetUpKafkaCheckers(ctx context.Context, requiredTopics []string, producer ProducerClient, intervalMS int, lag int,
	consumer ConsumerClient) *AsyncKafkaChecker {
	kafka := NewAsyncKafkaChecker(NewSyncKafkaChecker(SyncKafkaChecker{
		Name:           "kafka-health-check",
		Interval:       time.Duration(intervalMS) * time.Millisecond,
		Producer:       producer,
		Consumer:       consumer,
		RequiredTopics: requiredTopics,
		MaxLag:         int64(lag),
	}, NewConsumerOffsetStore()), 5, 1*time.Second)

	return kafka
}
