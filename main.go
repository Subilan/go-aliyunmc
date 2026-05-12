package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"go-aliyunmc/aliyun"
	"go-aliyunmc/env"
	"go-aliyunmc/h"
	"go-aliyunmc/log_util"
	"go-aliyunmc/monitors"
	"go-aliyunmc/perms"
	"go-aliyunmc/routes/bss_routes"
	"go-aliyunmc/routes/changelog_routes"
	"go-aliyunmc/routes/instance_routes"
	"go-aliyunmc/routes/monitor_routes"
	"go-aliyunmc/routes/sample_routes"
	"go-aliyunmc/routes/server_routes"
	"go-aliyunmc/routes/simple_routes"
	"go-aliyunmc/routes/state_routes"
	"go-aliyunmc/routes/task_routes"
	"go-aliyunmc/routes/user_routes"
	"go-aliyunmc/session"
	"go-aliyunmc/store"
	"go-aliyunmc/store/models"
	"go-aliyunmc/tasks"
	"go-aliyunmc/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	utils.MustBindConfigToml(&C, "main")
	store.MustInitialize()
	perms.MustInitialize()
	aliyun.MustInitialize()
	env.MustInitialize()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <command>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  run          启动服务器\n")
		fmt.Fprintf(os.Stderr, "  migrate      手动执行数据库迁移\n")
		fmt.Fprintf(os.Stderr, "  create_user  创建用户\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runServer()
	case "migrate":
		runMigrate()
	case "create_user":
		runCreateUser()
	default:
		fmt.Fprintf(os.Stderr, "未知指令: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runServer() {
	if env.DEV {
		if _, err := store.EnsureDevUser(); err != nil {
			log_util.Fatal("初始化DEV用户失败: %v", err)
		}
	}

	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	engine.Use(cors.New(C.Cors.GinCorsConfig()))

	engine.GET("/", h.V(func() string { return "Hello, World!" }))

	// 注册任务路由
	task_routes.Bind(engine)

	// 注册状态路由
	state_routes.Bind(engine)

	// 注册监控路由
	monitor_routes.Bind(engine)

	// 注册用户路由
	user_routes.Bind(engine)

	// 注册实例管理路由
	instance_routes.Bind(engine)

	// 注册服务器管理路由
	server_routes.Bind(engine)

	// 注册采样路由
	sample_routes.Bind(engine)

	// 注册财务账单路由
	bss_routes.Bind(engine)

	// 注册更新日志路由
	changelog_routes.Bind(engine)

	// 注册简单数据路由
	simple_routes.Bind(engine)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	monitors.MustInitialize(ctx)

	if !env.DEV {
		gin.SetMode(gin.ReleaseMode)
	}

	if env.DEV || !C.TLS.Enabled {
		run(engine, C.Expose, stop)
	} else {
		runTLS(engine, stop)
		run(engine, C.TLS.HttpPort, stop)
	}

	<-ctx.Done()
	log_util.Info("清理中...")

	log_util.Info("清除正在运行的任务")
	var wg sync.WaitGroup
	tasks.RangeExecutors(func(taskId uint, executor *tasks.Executor) {
		wg.Go(func() {
			executor.Interrupt()
			<-executor.Done()
		})
	})
	wg.Wait()
	log_util.Info("清除完毕")
}

func runMigrate() {
	store.AutoMigrate()
	log_util.Info("数据库迁移完成")
}

func runCreateUser() {
	fs := flag.NewFlagSet("create_user", flag.ExitOnError)
	username := fs.String("username", "", "用户名")
	password := fs.String("password", "", "密码")
	role := fs.String("role", "basic", "角色 (basic, operator, superuser)")
	fs.Parse(os.Args[2:])

	if *username == "" || *password == "" {
		fmt.Fprintf(os.Stderr, "错误: --username 和 --password 为必填项\n")
		os.Exit(1)
	}

	validRoles := map[string]perms.Role{
		"basic":     perms.RoleBasic,
		"operator":  perms.RoleOperator,
		"superuser": perms.RoleSuperuser,
	}
	r, ok := validRoles[*role]
	if !ok {
		fmt.Fprintf(os.Stderr, "错误: 无效角色 '%s'，可选值: basic, operator, superuser\n", *role)
		os.Exit(1)
	}

	var existing models.User
	if err := store.DB.Where("username = ?", *username).First(&existing).Error; err == nil {
		fmt.Fprintf(os.Stderr, "错误: 用户 '%s' 已存在\n", *username)
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 密码哈希失败: %v\n", err)
		os.Exit(1)
	}

	user := models.User{
		Username:     *username,
		PasswordHash: string(hash),
		Role:         r,
	}
	if err := store.DB.Create(&user).Error; err != nil {
		fmt.Fprintf(os.Stderr, "错误: 创建用户失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("用户 '%s' 创建成功，角色: %s\n", *username, *role)
}

func runTLS(engine *gin.Engine, cancel context.CancelFunc) {
	go func() {
		if err := http.ListenAndServeTLS(fmt.Sprintf(":%d", C.Expose), C.TLS.CertFile, C.TLS.KeyFile, session.LoadAndSave(engine)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
			cancel()
		}
	}()
}

func run(engine *gin.Engine, port int, cancel context.CancelFunc) {
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), session.LoadAndSave(engine)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
			cancel()
		}
	}()
}
