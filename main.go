package main

import (
	"context"
	"errors"
	"fmt"
	"go-aliyunmc/aliyun"
	"go-aliyunmc/casbin"
	"go-aliyunmc/env"
	"go-aliyunmc/routes/task_routes"
	"go-aliyunmc/routes/user_routes"
	"go-aliyunmc/store"
	"go-aliyunmc/tasks"
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
	// 加载配置
	MustLoadConfig()
	store.MustLoadConfig()
	casbin.MustLoadConfig()
	aliyun.MustLoadConfig()

	// 初始化模块
	store.MustInitialize()
	casbin.MustInitialize()
	aliyun.MustInitialize()
	env.MustInitialize()
	tasks.MustInitialize()

	if env.DEV {
		store.AutoMigrate()
	}

	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	// 配置session中间件
	engine.Use(sessions.Sessions("session", C.Session.GetSessionStore()))

	engine.Use(cors.New(C.Cors.GinCorsConfig()))

	// 注册任务路由
	task_routes.Bind(engine)

	// 注册用户路由
	user_routes.Bind(engine)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if !env.DEV {
		gin.SetMode(gin.ReleaseMode)
	}

	if env.DEV || !C.Autotls.Enabled {
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
		if err := autotls.RunWithContext(ctx, engine, C.Autotls.Domains...); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
			cancel()
		}
	}()
}

func run(engine *gin.Engine, cancel context.CancelFunc) {
	go func() {
		if err := engine.Run(fmt.Sprintf(":%d", C.Expose)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
			cancel()
		}
	}()
}
