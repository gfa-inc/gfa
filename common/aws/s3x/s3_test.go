package s3x

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	config.Setup(config.WithPath("../../../"))
	logger.Setup()
	Setup()
	assert.NotNil(t, Client)
	assert.NotNil(t, GetClient("default"))

	object, err := Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(config.GetString("aws.s3.default.bucket")),
		Key:    aws.String("test.txt"),
		Body:   strings.NewReader("Hello, S3!"),
	})
	assert.Nil(t, err)
	assert.NotNil(t, object)
}
