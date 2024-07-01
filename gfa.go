package gfa

import (
	"context"
	"errors"
	"github.com/gfa-inc/gfa/common/cache"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/db"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/common/nsdb"
	"github.com/gfa-inc/gfa/common/swag"
	"github.com/gfa-inc/gfa/core"
	"github.com/gfa-inc/gfa/middlewares"
	"github.com/gfa-inc/gfa/middlewares/security"
	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"log"
	"os/signal"
	"strings"
	"syscall"
)

var (
	gfa Gfa
)

type Gfa struct {
	*gin.Engine
	mdws []gin.HandlerFunc

	controllers []core.Controller
	ginOpts     []gin.OptionFunc
	cfgOpts     []config.OptionFunc
}

func NewGfa() Gfa {
	return Gfa{
		controllers: make([]core.Controller, 0),
	}
}

func (g *Gfa) WithGinOption(opts ...gin.OptionFunc) {
	g.ginOpts = append(g.ginOpts, opts...)
}

func (g *Gfa) WithCfgOption(opts ...config.OptionFunc) {
	g.cfgOpts = append(g.cfgOpts, opts...)
}

func (g *Gfa) WithMiddleware(mdws ...gin.HandlerFunc) {
	g.mdws = append(g.mdws, mdws...)
}

func (g *Gfa) Run() {
	config.Setup(g.cfgOpts...)

	logger.Setup(parseLoggerConfig())

	cache.Setup()

	db.Setup()

	nsdb.Setup()

	if !logger.IsDebugLevelEnabled() {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.DebugPrintFunc = func(format string, values ...interface{}) {
		logger.Debugf(strings.TrimSpace(format), values...)
	}
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		logger.Debugf("%s %s %s %d", httpMethod, absolutePath, handlerName, nuHandlers)
	}

	g.Engine = gin.New()
	// recovery
	g.Use(gin.Recovery())
	// onerror
	g.Use(middlewares.OnError())
	// requestid
	g.Use(middlewares.Requestid())
	// access log
	g.Use(middlewares.AccessLog())
	// custom middlewares
	for _, mdw := range g.mdws {
		g.Use(mdw)
	}
	// session
	if middlewares.SessionEnabled() {
		g.Use(middlewares.Session())
	}
	// security
	g.Use(security.Security())

	basePath := config.GetString("server.base_path")
	rootRouter := g.Group(basePath)
	// swagger
	swag.Setup(rootRouter)
	// routes
	for _, controller := range g.controllers {
		controller.Setup(rootRouter)
	}

	addr := config.GetString("server.addr")

	gracefulServer, err := graceful.New(g.Engine, graceful.WithAddr(addr))
	if err != nil {
		logger.Error(err)
		return
	}
	defer gracefulServer.Close()

	// graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Infof("Listen and Serving HTTP on http://%s", addr)
	err = gracefulServer.RunWithContext(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Panic(err)
	}
}

func parseLoggerConfig() logger.Config {
	var option logger.Config
	err := config.UnmarshalKey("logger", &option)
	if err != nil {
		log.Panic(err)
	}
	return option
}

func WithGinOption(opts ...gin.OptionFunc) {
	gfa.WithGinOption(opts...)
}

func WithCfgOption(opts ...config.OptionFunc) {
	gfa.WithCfgOption(opts...)
}

func WithMiddleware(mdws ...gin.HandlerFunc) {
	gfa.WithMiddleware(mdws...)
}

func Default() {
	gfa = NewGfa()
}

func Run() {
	printBanner()
	gfa.Run()
}

func AddController(controller core.Controller) {
	gfa.controllers = append(gfa.controllers, controller)
}

func AddControllers(controllers ...core.Controller) {
	gfa.controllers = append(gfa.controllers, controllers...)
}

func PermitRoute(route string) {
	security.PermitRoute(route)
}

func PermitRoutes(routes []string) {
	security.PermitRoutes(routes)
}
