package kafkax

import (
	"context"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetup(t *testing.T) {
	config.Setup(config.WithPath("../../../../"))
	logger.Setup()
	Setup()
	assert.NotNil(t, ConsumerClient)
	assert.NotNil(t, ProducerClient)
	assert.Greater(t, len(consumerClientPool), 0)
	assert.Greater(t, len(producerClientPool), 0)
}

func TestDial(t *testing.T) {
	conn, err := kafka.DialContext(context.Background(), "tcp", "localhost:9092")
	assert.Nil(t, err)
	defer conn.Close()

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             "gfa",
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	assert.Nil(t, err)

	defer func() {
		err = conn.DeleteTopics("gfa")
		assert.Nil(t, err)
	}()

	writer := &kafka.Writer{
		Addr:                   kafka.TCP("localhost:9092"),
		Topic:                  "gfa",
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
	}
	defer writer.Close()

	err = writer.WriteMessages(context.Background(), []kafka.Message{
		{
			Value: []byte("hello gfa!"),
		},
	}...)
	assert.Nil(t, err)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{"localhost:9092"},
		Topic:     "gfa",
		Partition: 0,
		//GroupID:   "gfa",
	})
	defer reader.Close()

	for {
		m, err := reader.ReadMessage(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, "hello gfa!", string(m.Value))
		break
	}
}
