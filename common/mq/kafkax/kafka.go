package kafkax

import (
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/utils/ptr"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	"log"
)

var (
	ConsumerClient     *kafka.Reader
	consumerClientPool map[string]*kafka.Reader
	ProducerClient     *kafka.Writer
	producerClientPool map[string]*kafka.Writer
)

type Config struct {
	Type string

	ConsumerConfig `mapstructure:",squash"`
	ProducerConfig `mapstructure:",squash"`
}

type ConsumerConfig struct {
	Brokers     []string
	Topic       string
	GroupID     string
	GroupTopics []string
	Partition   int
	Mechanism   string
	Username    string
	Password    string
	Default     bool
}

type ProducerConfig struct {
	Brokers   []string
	Topic     string
	Async     *bool
	Mechanism string
	Username  string
	Password  string
}

func NewConsumerClient(option ConsumerConfig) *kafka.Reader {
	cfg := kafka.ReaderConfig{
		Brokers:     option.Brokers,
		Topic:       option.Topic,
		GroupID:     option.GroupID,
		GroupTopics: option.GroupTopics,
		Partition:   option.Partition,
		ErrorLogger: kafka.LoggerFunc(logger.Errorf),
	}

	cfg.Dialer = fillMechanism(option.Mechanism, option.Username, option.Password)

	reader := kafka.NewReader(cfg)

	return reader
}

func NewProducerClient(option ProducerConfig) *kafka.Writer {
	if option.Async == nil {
		option.Async = ptr.To(true)
	}
	cfg := kafka.WriterConfig{
		Brokers:     option.Brokers,
		Topic:       option.Topic,
		Balancer:    &kafka.LeastBytes{},
		ErrorLogger: kafka.LoggerFunc(logger.Errorf),
		Async:       *option.Async,
	}

	cfg.Dialer = fillMechanism(option.Mechanism, option.Username, option.Password)

	writer := kafka.NewWriter(cfg)

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

func fillMechanism(mechanism string, username string, password string) *kafka.Dialer {
	dialer := kafka.DefaultDialer

	switch mechanism {
	case "PLAIN":
		dialer.SASLMechanism = plain.Mechanism{
			Username: username,
			Password: password,
		}
	case "SCRAM-SHA-256":
		var err error
		dialer.SASLMechanism, err = scram.Mechanism(scram.SHA256, username, password)
		if err != nil {
			logger.Panicf("Failed to create SCRAM-SHA-256 mechanism: %s", err)
		}
	case "SCRAM-SHA-512":
		var err error
		dialer.SASLMechanism, err = scram.Mechanism(scram.SHA512, username, password)
		if err != nil {
			logger.Panicf("Failed to create SCRAM-SHA-512 mechanism: %s", err)
		}
	}

	return dialer
}
