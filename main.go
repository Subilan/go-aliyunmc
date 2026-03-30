package main

import (
	"context"
	"errors"
	"fmt"
	"go-aliyunmc/aliyun/clients"
	"go-aliyunmc/casbin"
	"go-aliyunmc/config"
	"go-aliyunmc/globals"
	"go-aliyunmc/routes/user_routes"
	"go-aliyunmc/store"
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
	config.MustInitialize()
	globals.MustInitialize()
	store.MustInitialize(config.G.Store)
	casbin.MustInitialize()
	clients.MustInitialize()

	if globals.DEV {
		store.AutoMigrate()
	}

	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	// 配置session中间件
	engine.Use(sessions.Sessions("session", config.G.Base.Session.GetSessionStore()))

	engine.Use(cors.New(config.G.Base.Cors.GinCorsConfig()))

	// 注册用户路由
	user_routes.Bind(engine)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if !globals.DEV {
		gin.SetMode(gin.ReleaseMode)
	}

	if globals.DEV || !config.G.Base.Autotls.Enabled {
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
		if err := autotls.RunWithContext(ctx, engine, config.G.Base.Autotls.Domains...); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
			cancel()
		}
	}()
}

func run(engine *gin.Engine, cancel context.CancelFunc) {
	go func() {
		if err := engine.Run(fmt.Sprintf(":%d", config.G.Base.Expose)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
			cancel()
		}
	}()
}
