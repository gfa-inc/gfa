package kafkax

import (
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/utils/ptr"
	"github.com/segmentio/kafka-go"
	"log"
)

var (
	ConsumerClient     *kafka.Reader
	consumerClientPool map[string]*kafka.Reader
	ProducerClient     *kafka.Writer
	producerClientPool map[string]*kafka.Writer
)

type Config struct {
	Type           string `json:"type"`
	ConsumerConfig `mapstructure:",squash"`
	ProducerConfig `mapstructure:",squash"`
}

type ConsumerConfig struct {
	Brokers   []string
	Topic     string
	GroupID   string
	Partition int
	Default   bool
}

func NewConsumerClient(option ConsumerConfig) *kafka.Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     option.Brokers,
		Topic:       option.Topic,
		GroupID:     option.GroupID,
		Partition:   option.Partition,
		ErrorLogger: kafka.LoggerFunc(logger.Errorf),
	})

	return reader
}

type ProducerConfig struct {
	Brokers []string
	Topic   string
	Async   *bool
}

func NewProducerClient(option ProducerConfig) *kafka.Writer {
	if option.Async == nil {
		option.Async = ptr.ToBool(true)
	}
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:     option.Brokers,
		Topic:       option.Topic,
		Balancer:    &kafka.LeastBytes{},
		ErrorLogger: kafka.LoggerFunc(logger.Errorf),
		Async:       *option.Async,
	})

	return writer
}

func Setup() {
	consumerClientPool = make(map[string]*kafka.Reader)
	producerClientPool = make(map[string]*kafka.Writer)

	if config.Get("kafka") == nil {
		logger.Debug("No kafka config found")
		return
	}

	configMap := make(map[string]Config)
	err := config.UnmarshalKey("kafka", &configMap)
	if err != nil {
		log.Panic(err)
	}

	for k, v := range configMap {
		if v.Type == "consumer" || v.Type == "" {
			client := NewConsumerClient(v.ConsumerConfig)
			PutConsumerClient(k, client)

			if v.Default {
				ConsumerClient = client
			}
		}
		if v.Type == "producer" || v.Type == "" {
			client := NewProducerClient(v.ProducerConfig)
			PutProducerClient(k, client)

			if v.Default {
				ProducerClient = client
			}
		}
	}

	logger.Infof("Success to init kafka, %d consumer and %d producer", len(consumerClientPool), len(producerClientPool))
}

func GetConsumerClient(name string) *kafka.Reader {
	client, ok := consumerClientPool[name]
	if !ok {
		logger.Panicf("Kafka Consumer %s not found", name)
	}
	return client
}

func PutConsumerClient(name string, client *kafka.Reader) {
	consumerClientPool[name] = client
}

func GetProducerClient(name string) *kafka.Writer {
	client, ok := producerClientPool[name]
	if !ok {
		logger.Panicf("Kafka Producer %s not found", name)
	}
	return client
}

func PutProducerClient(name string, client *kafka.Writer) {
	producerClientPool[name] = client
}

func HasConsumerClient(name string) bool {
	_, ok := consumerClientPool[name]
	return ok
}

func HasProducerClient(name string) bool {
	_, ok := producerClientPool[name]
	return ok
}
