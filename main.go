package main

import (
	"context"
	"errors"
	"fmt"
	"go-aliyunmc-v2/casbin"
	"go-aliyunmc-v2/config"
	"go-aliyunmc-v2/globals"
	"go-aliyunmc-v2/routes/user_routes"
	"go-aliyunmc-v2/store"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
)

func main() {
	config.MustBindGlobal()
	globals.MustLoadGlobals()
	store.MustInitialize(config.Global.Store)
	casbin.MustInitialize()

	if globals.DEV {
		store.AutoMigrate()
	}

	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	// 配置session中间件
	engine.Use(sessions.Sessions("session", config.Global.Base.Session.GetSessionStore()))

	engine.Use(cors.New(config.Global.Base.Cors.GinCorsConfig()))

	// 注册用户路由
	user_routes.Bind(engine)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if !globals.DEV {
		gin.SetMode(gin.ReleaseMode)
	}

	if globals.DEV || !config.Global.Base.Autotls.Enabled {
		run(engine, stop)
	} else {
		runTLS(engine, ctx, stop)
	}

	<-ctx.Done()
	log.Println("关闭中...")
	// cleanup logic here
}

func runTLS(engine *gin.Engine, ctx context.Context, cancel context.CancelFunc) {
	go func() {
		if err := autotls.RunWithContext(ctx, engine, config.Global.Base.Autotls.Domains...); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
			cancel()
		}
	}()
}

func run(engine *gin.Engine, cancel context.CancelFunc) {
	go func() {
		if err := engine.Run(fmt.Sprintf(":%d", config.Global.Base.Expose)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
			cancel()
		}
	}()
}
