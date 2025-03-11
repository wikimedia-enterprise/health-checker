package health

import (
	"sync"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// Store previous offsets for each partition
type ConsumerOffsetStore struct {
	positions map[string]map[int32]kafka.Offset // topic -> partition -> current position
	commits   map[string]map[int32]kafka.Offset // topic -> partition -> committed offset
	mu        sync.RWMutex
}

func NewConsumerOffsetStore() *ConsumerOffsetStore {
	return &ConsumerOffsetStore{
		positions: make(map[string]map[int32]kafka.Offset),
		commits:   make(map[string]map[int32]kafka.Offset),
	}
}

func (s *ConsumerOffsetStore) GetOffsets(topic string, partition int32) (position kafka.Offset, commit kafka.Offset, exists bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	positions, okPos := s.positions[topic]
	commits, okCom := s.commits[topic]
	if !okPos || !okCom {
		return 0, 0, false
	}

	position, okPos = positions[partition]
	commit, okCom = commits[partition]
	if !okPos || !okCom {
		return 0, 0, false
	}

	return position, commit, true
}

func (s *ConsumerOffsetStore) SetOffsets(topic string, partition int32, position kafka.Offset, commit kafka.Offset) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.positions[topic]; !ok {
		s.positions[topic] = make(map[int32]kafka.Offset)
	}
	if _, ok := s.commits[topic]; !ok {
		s.commits[topic] = make(map[int32]kafka.Offset)
	}

	s.positions[topic][partition] = position
	s.commits[topic][partition] = commit
}

func (s *ConsumerOffsetStore) RevokePartitions(revoked []kafka.TopicPartition) {
	for _, partition := range revoked {
		if partition.Topic == nil {
			continue
		}
		if topic, exists := s.positions[*partition.Topic]; exists {
			delete(topic, partition.Partition)
		}
		if topic, exists := s.commits[*partition.Topic]; exists {
			delete(topic, partition.Partition)
		}
	}
}
