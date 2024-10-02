package elasticx

import (
	"context"
	"crypto/tls"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/samber/lo"
	"net/http"
	"strings"
	"time"
)

var (
	Client     *elasticsearch.TypedClient
	clientPool map[string]*elasticsearch.TypedClient
)

type Config struct {
	Name     string
	Addrs    []string
	Username string
	Password string
	Default  bool
}

func NewClient(option Config) (*elasticsearch.TypedClient, error) {
	client, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses:           option.Addrs,
		Username:            option.Username,
		Password:            option.Password,
		CompressRequestBody: true,
		EnableDebugLogger:   logger.IsDebugLevelEnabled(),
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	})
	if err != nil {
		logger.Panicf("Fail to connect elastic, %s", err)
		return nil, err
	}

	pingTimeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = client.Ping().Do(pingTimeout)
	if err != nil {
		logger.Warnf("Fail to ping elastic, %s", err)
		return nil, nil
	}

	logger.Infof("Connectng to elastic [%s] %s", option.Name, strings.Join(option.Addrs, ", "))
	return client, nil
}

func Setup() {
	clientPool = make(map[string]*elasticsearch.TypedClient)

	if config.Get("elastic") == nil {
		logger.Debug("No elastic config found")
		return
	}

	configMap := make(map[string]Config)
	err := config.UnmarshalKey("elastic", &configMap)
	if err != nil {
		logger.Panic(err)
	}

	logger.Info("Starting to initialize elastic client pool")
	for name, option := range configMap {
		option.Name = name
		client, err := NewClient(option)
		if err != nil {
			logger.Panic(err)
		}
		PutClient(name, client)

		if option.Default {
			Client = client
		}
	}

	logger.Infof("Elastic client pool has been initialized with %d clients, clients: %s",
		len(clientPool), strings.Join(lo.Keys(clientPool), ", "))
}

func GetClient(name string) *elasticsearch.TypedClient {
	client, ok := clientPool[name]
	if !ok {
		logger.Panicf("Elastic Client %s not found", name)
	}
	return client
}

func PutClient(name string, client *elasticsearch.TypedClient) {
	clientPool[name] = client
}
