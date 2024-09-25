package s3x

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/duke-git/lancet/v2/maputil"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"strings"
)

var (
	Client     *s3.Client
	clientPool map[string]*s3.Client
)

type Config struct {
	Name     string
	Endpoint string
	AK       string
	SK       string
	Region   string
	Bucket   string
	Default  bool
}

func NewS3Client(option Config) (*s3.Client, error) {
	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(option.Region),
		awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(option.AK, option.SK, "")),
		awsConfig.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               option.Endpoint,
				HostnameImmutable: true,
			}, nil
		})),
	)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	client := s3.NewFromConfig(cfg)
	logger.Debugf("Connecting to s3 [%s] %s", option.Name, option.Bucket)
	return client, nil
}

func Setup() {
	clientPool = make(map[string]*s3.Client)

	if config.Get("aws.s3") == nil {
		logger.Debug("No aws s3 config found")
		return
	}

	configMap := make(map[string]Config)
	err := config.UnmarshalKey("aws.s3", &configMap)
	if err != nil {
		logger.Panic(err)
		return
	}

	logger.Infof("Starting to initialize aws s3 client pool")
	for k, v := range configMap {
		v.Name = k
		var client *s3.Client
		client, err = NewS3Client(v)
		if err != nil {
			logger.Error(err)
			continue
		}
		PutClient(k, client)

		if v.Default {
			Client = client
		}
	}

	logger.Infof("Aws s3 client pool has been initialized with %d clients, clients: %s",
		len(clientPool), strings.Join(maputil.Keys(clientPool), ", "))
}

func GetClient(name string) *s3.Client {
	client, ok := clientPool[name]
	if !ok {
		logger.Panicf("Aws s3 client %s not found", name)
	}
	return client
}

func PutClient(name string, client *s3.Client) {
	clientPool[name] = client
}
