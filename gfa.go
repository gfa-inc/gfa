package gfa

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/gfa-inc/gfa/common/aws"
	"github.com/gfa-inc/gfa/common/cache"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/db"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/common/messenger/mailx"
	"github.com/gfa-inc/gfa/common/mq"
	"github.com/gfa-inc/gfa/common/nsdb"
	"github.com/gfa-inc/gfa/common/swag"
	"github.com/gfa-inc/gfa/common/validatorx"
	"github.com/gfa-inc/gfa/core"
	"github.com/gfa-inc/gfa/middlewares"
	"github.com/gfa-inc/gfa/middlewares/accesslog"
	"github.com/gfa-inc/gfa/middlewares/requestid"
	"github.com/gfa-inc/gfa/middlewares/security"
	"github.com/gfa-inc/gfa/middlewares/session"
	"github.com/gfa-inc/gfa/utils"
	"github.com/gfa-inc/gfa/utils/syncx"
	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
)

var (
	gfa       Gfa
	cancelCtx context.Context
	cancelWg  sync.WaitGroup
)

type Gfa struct {
	Engine     *gin.Engine
	mdws       []gin.HandlerFunc
	setups     []func()
	postSetups []func()

	controllers []core.Controller
	ginOpts     []gin.OptionFunc
	cfgOpts     []config.OptionFunc
}

func NewGfa() Gfa {
	return Gfa{
		controllers: make([]core.Controller, 0),
	}
}

func init() {
	gfa = NewGfa()
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

func (g *Gfa) WithSetup(setup func()) {
	g.setups = append(g.setups, setup)
}

func (g *Gfa) WithPostSetup(setup func()) {
	g.postSetups = append(g.postSetups, setup)
}

func (g *Gfa) Run() {
	addr := config.GetString("server.addr")

	gracefulServer, err := graceful.New(g.Engine, graceful.WithAddr(addr))
	if err != nil {
		logger.Error(err)
		return
	}
	defer gracefulServer.Close()

	var stop context.CancelFunc
	// graceful shutdown
	cancelCtx, stop = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	syncx.All(func() {
		logger.Infof("Listen and Serving HTTP on http://%s", addr)
		err = gracefulServer.RunWithContext(cancelCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Panic(err)
		}
	}, func() {
		cancelWg.Wait()
	})
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

func WithSetup(setups ...func()) {
	for _, setup := range setups {
		gfa.WithSetup(setup)
	}
}

func WithPostSetup(setups ...func()) {
	for _, setup := range setups {
		gfa.WithPostSetup(setup)
	}
}

func ParseLoggerConfig() logger.OptionFunc {
	return func(option *logger.Config) {
		err := config.UnmarshalKey("logger", option)
		if err != nil {
			log.Panic(err)
		}
	}
}

func Default() *Gfa {
	config.Setup(gfa.cfgOpts...)

	printBanner()

	logger.Setup(ParseLoggerConfig())

	for _, setup := range gfa.setups {
		setup()
	}

	cache.Setup()

	db.Setup()

	nsdb.Setup()

	mq.Setup()

	validatorx.Setup()

	aws.Setup()

	mailx.Setup()

	// post setups
	for _, setup := range gfa.postSetups {
		setup()
	}

	if !logger.IsDebugLevelEnabled() {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.DebugPrintFunc = func(format string, values ...interface{}) {
		logger.Debugf(strings.TrimSpace(format), values...)
	}
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		logger.Debugf("%s %s %s %d", httpMethod, absolutePath, handlerName, nuHandlers)
	}

	gfa.Engine = gin.New(gfa.ginOpts...)
	// recovery
	gfa.Engine.Use(gin.Recovery())
	// onerror
	gfa.Engine.Use(middlewares.OnError())
	// requestid
	gfa.Engine.Use(requestid.RequestID())
	// access log
	gfa.Engine.Use(accesslog.AccessLog())
	// session
	if session.Enabled() {
		gfa.Engine.Use(session.Session())
	}
	// security
	if security.Enabled() {
		gfa.Engine.Use(security.Security())
	}
	// custom middlewares
	for _, mdw := range gfa.mdws {
		gfa.Engine.Use(mdw)
	}

	return &gfa
}

func Run() {
	basePath := config.GetString("server.base_path")
	rootRouter := gfa.Engine.Group(basePath)
	// swagger
	swag.Setup(rootRouter)
	// custom routes
	for _, controller := range gfa.controllers {
		controller.Setup(rootRouter)
	}

	gfa.Run()
}

func AddController(controller core.Controller) {
	gfa.controllers = append(gfa.controllers, controller)
}

func AddControllers(controllers ...core.Controller) {
	gfa.controllers = append(gfa.controllers, controllers...)
}

type GroupController struct {
	Group       string
	Controllers []core.Controller
}

func (gc *GroupController) Setup(r *gin.RouterGroup) {
	gr := r.Group(gc.Group)
	for _, controller := range gc.Controllers {
		controller.Setup(gr)
	}
}

func AddGroupControllers(group string, controllers ...core.Controller) {
	gfa.controllers = append(gfa.controllers, &GroupController{
		Group:       group,
		Controllers: controllers,
	})
}

// Async runs a function asynchronously and tracks its completion for graceful shutdown.
func Async(fn func()) {
	cancelWg.Add(1)
	go func() {
		defer cancelWg.Done()
		fn()
	}()
}

// AsyncWithCancel runs a function asynchronously with a cancelable context and tracks its completion for graceful shutdown.
func AsyncWithCancel(fn func(cancelCtx context.Context)) {
	cancelWg.Add(1)
	go func() {
		defer cancelWg.Done()
		fn(cancelCtx)
	}()
}

func Test(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	roots := utils.FindModuleRoot(wd)
	if roots == "" {
		t.Fatal("cannot find module root")
	}

	config.Setup(config.WithPath(roots), config.WithPath(roots+"/configs"))

	logger.Setup(ParseLoggerConfig())
}
