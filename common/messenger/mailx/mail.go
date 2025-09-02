package mailx

import (
	"strings"

	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/samber/lo"
	"github.com/wneessen/go-mail"
)

var (
	clientPool map[string]*mail.Client
	Client     *mail.Client
)

type Config struct {
	Name     string
	Host     string
	Port     int
	Username string
	Password string
	SMTP     struct {
		AuthType string
	}
	SSL struct {
		Enable bool
		Port   int
	}
	Default bool
}

func NewClient(option Config) (*mail.Client, error) {
	var opts []mail.Option
	if option.Port != 0 {
		opts = append(opts, mail.WithPort(option.Port))
	}
	if option.Username != "" {
		opts = append(opts, mail.WithUsername(option.Username))
	}
	if option.Password != "" {
		opts = append(opts, mail.WithPassword(option.Password))
	}
	if option.SMTP.AuthType != "" {
		var authType mail.SMTPAuthType
		err := authType.UnmarshalString(option.SMTP.AuthType)
		if err != nil {
			return nil, err
		}
		opts = append(opts, mail.WithSMTPAuth(authType))
	} else {
		opts = append(opts, mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover))
	}

	if option.SSL.Enable {
		opts = append(opts, mail.WithSSL())
		if option.SSL.Port != 0 {
			opts = append(opts, mail.WithPort(option.SSL.Port))
		}
	}

	client, err := mail.NewClient(option.Host, opts...)
	if err != nil {
		logger.Errorf("fail to new client %s, %s", option.Name, err)
		return nil, err
	}

	return client, nil
}

func PutClient(name string, client *mail.Client) {
	clientPool[name] = client
}

func Setup() {
	clientPool = make(map[string]*mail.Client)

	if config.Get("mail") == nil {
		logger.Debug("No mail config found")
		return
	}

	configMap := make(map[string]Config)
	err := config.UnmarshalKey("mail", &configMap)
	if err != nil {
		logger.Panic(err)
	}

	logger.Infof("Starting to initialize mail client pool")
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

	logger.Infof("Mail client pool has been initialized with %d clients, clients: %s",
		len(clientPool), strings.Join(lo.Keys(clientPool), ", "))
}
