package mysqlx

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/samber/lo"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	glog "gorm.io/gorm/logger"
)

var (
	Client     *gorm.DB
	clientPool map[string]*gorm.DB
)

type Config struct {
	Name            string
	DNS             string
	Level           string
	ConnMaxLifeTime int
	MaxIdleConns    int
	Default         bool
}

func NewClient(option Config) (client *gorm.DB, err error) {
	dial := mysql.Open(option.DNS)
	mysqlDial, _ := dial.(*mysql.Dialector)
	client, err = gorm.Open(dial, &gorm.Config{
		Logger:      newGormLogger(option),
		PrepareStmt: false,
	})
	if err != nil {
		logger.Error(err)
		return
	}

	var db *sql.DB
	db, err = client.DB()
	if err != nil {
		logger.Error(err)
		return
	}

	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxIdleConns(10)

	logger.Infof("Connecting to mysql [%s] %s", option.Name, mysqlDial.Config.DSNConfig.Addr)
	return
}

func Setup() {
	clientPool = make(map[string]*gorm.DB)

	if config.Get("mysql") == nil {
		logger.Debug("No mysql config found")
		return
	}

	configMap := make(map[string]Config)
	err := config.UnmarshalKey("mysql", &configMap)
	if err != nil {
		logger.Panic(err)
	}

	logger.Infof("Starting to initialize mysql client pool")
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

	logger.Infof("Mysql client pool has been initialized with %d clients, clients: %s",
		len(clientPool), strings.Join(lo.Keys(clientPool), ", "))
}

func GetClient(name string) *gorm.DB {
	client, ok := clientPool[name]
	if !ok {
		logger.Panicf("Mysql Client %s not found", name)
	}
	return client
}

func PutClient(name string, client *gorm.DB) {
	clientPool[name] = client
}

func newGormLogger(config Config) *gormLogger {
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		logger.Panic(err)
	}

	l := logger.GetGlobalLogger().Clone(level)
	return &gormLogger{
		Logger: l,
	}
}

type gormLogger struct {
	logger.Logger
}

func (g *gormLogger) LogMode(logLevel glog.LogLevel) glog.Interface {
	var level zapcore.Level
	switch logLevel {
	case glog.Warn:
		level = zapcore.WarnLevel
	case glog.Error:
		level = zapcore.ErrorLevel
	case glog.Info:
		fallthrough
	default:
		level = zapcore.DebugLevel
	}

	ng := gormLogger{
		Logger: g.Clone(level),
	}
	return &ng
}

func (g *gormLogger) Info(c context.Context, format string, args ...any) {
	g.Infof(c, format, args)
}

func (g *gormLogger) Warn(c context.Context, format string, args ...any) {
	g.Warnf(c, format, args)
}

func (g *gormLogger) Error(c context.Context, format string, args ...any) {
	g.Errorf(c, format, args)
}

func (g *gormLogger) Trace(c context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if !g.LevelEnabled("debug") {
		return
	}
	sqlContent, rowsAffected := fc()
	elapsed := time.Since(begin)
	g.Debugf(c, "[%dms] [rows:%d] %s", elapsed.Milliseconds(), rowsAffected, sqlContent)
}

func AssignmentNotEmptyColumns(names []string) clause.Set {
	values := make(map[string]any)
	for _, name := range names {
		values[name] = fmt.Sprintf("CASE WHEN VALUES(%s) IS NOT NULL AND TRIM(VALUES(%s)) != '' THEN VALUES(%s) ELSE %s END",
			name, name, name, name)
	}

	return clause.Assignments(values)
}
