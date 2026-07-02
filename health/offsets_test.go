package health

import (
	"sync"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/suite"
)

// ConsumerOffsetStoreTestSuite defines the test suite for ConsumerOffsetStore
type ConsumerOffsetStoreTestSuite struct {
	suite.Suite
	store *ConsumerOffsetStore
}

func (suite *ConsumerOffsetStoreTestSuite) SetupTest() {
	suite.store = NewConsumerOffsetStore()
}

func (suite *ConsumerOffsetStoreTestSuite) TestSetAndGetOffsets() {
	topic := "test-topic"
	partition := int32(1)
	position := kafka.Offset(100)
	commit := kafka.Offset(90)

	suite.store.SetOffsets(topic, partition, position, commit)
	storedPosition, storedCommit, exists := suite.store.GetOffsets(topic, partition)

	suite.True(exists)
	suite.Equal(position, storedPosition)
	suite.Equal(commit, storedCommit)
}

func (suite *ConsumerOffsetStoreTestSuite) TestGetOffsetsNonExistent() {
	topic := "non-existent-topic"
	partition := int32(1)

	position, commit, exists := suite.store.GetOffsets(topic, partition)

	suite.False(exists)
	suite.Equal(kafka.Offset(0), position)
	suite.Equal(kafka.Offset(0), commit)
}

func (suite *ConsumerOffsetStoreTestSuite) TestConcurrentAccess() {
	topic := "concurrent-topic"
	partition := int32(2)
	position := kafka.Offset(150)
	commit := kafka.Offset(140)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		suite.store.SetOffsets(topic, partition, position, commit)
	}()

	go func() {
		defer wg.Done()
		_, _, _ = suite.store.GetOffsets(topic, partition)
	}()

	wg.Wait()

	storedPosition, storedCommit, exists := suite.store.GetOffsets(topic, partition)

	suite.True(exists)
	suite.Equal(position, storedPosition)
	suite.Equal(commit, storedCommit)
}

func TestConsumerOffsetStoreTestSuite(t *testing.T) {
	suite.Run(t, new(ConsumerOffsetStoreTestSuite))
}
