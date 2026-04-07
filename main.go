package main

import (
	"context"
	"errors"
	"fmt"
	"go-aliyunmc/aliyun"
	"go-aliyunmc/casbin"
	"go-aliyunmc/env"
	"go-aliyunmc/logs"
	"go-aliyunmc/routes/instance_routes"
	"go-aliyunmc/routes/server_routes"
	"go-aliyunmc/routes/task_routes"
	"go-aliyunmc/routes/user_routes"
	"go-aliyunmc/server"
	"go-aliyunmc/store"
	"go-aliyunmc/tasks"
	"log"
	"net/http"
	"os/signal"
	"sync"
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
	server.MustLoadConfig()
	tasks.MustLoadConfig()

	// 初始化模块
	store.MustInitialize()
	casbin.MustInitialize()
	aliyun.MustInitialize()
	env.MustInitialize()
	if env.DEV {
		if _, err := store.EnsureDevUser(); err != nil {
			logs.Fatal("初始化DEV用户失败: %v", err)
		}
	}
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

	// 注册实例管理路由
	instance_routes.Bind(engine)

	// 注册服务器管理路由
	server_routes.Bind(engine)

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
	logs.Info("清理中...")
	// cleanup logic here

	logs.Info("清除正在运行的任务")
	var wg sync.WaitGroup
	tasks.RangeExecutors(func(taskId uint, executor *tasks.Executor) {
		wg.Go(func() {
			executor.Interrupt()
			<-executor.Done()
		})
	})
	wg.Wait()
	logs.Info("清除完毕")
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
