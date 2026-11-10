package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/env"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
	"github.com/Subilan/go-aliyunmc/internal/common/session_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum"
	forumsession "github.com/Subilan/go-aliyunmc/internal/modules/forum/session"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/monitors"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/policygen"
	labsession "github.com/Subilan/go-aliyunmc/internal/modules/lab/session"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/tasks"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	env.MustInitialize()

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <command>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  run          启动服务器\n")
		fmt.Fprintf(os.Stderr, "  migrate      手动执行数据库迁移\n")
		fmt.Fprintf(os.Stderr, "  create_user  创建用户\n")
		fmt.Fprintf(os.Stderr, "  check_config 校验配置，不连接任何外部依赖\n")
		fmt.Fprintf(os.Stderr, "  gen_policy   根据策略配置渲染 RAM 策略并输出到 stdout\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runServer()
	case "migrate":
		runMigrate()
	case "create_user":
		runCreateUser()
	case "check_config":
		runCheckConfig()
	case "gen_policy":
		runGenPolicy()
	default:
		fmt.Fprintf(os.Stderr, "未知指令: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runServer() {
	// 读入、填默认值并校验全部配置。
	config.MustLoad()

	store.MustInitialize()
	if err := store.AutoMigrate(modules.Entities()...); err != nil {
		log_util.Fatal("初始化开发数据库失败: %v", err)
	}
	if err := session_util.InitStore(store.DB, labsession.Manager, forumsession.Manager); err != nil {
		log_util.Fatal("初始化 session 存储失败: %v", err)
	}
	lab.MustInitialize()
	forum.MustInitialize()

	if env.DEV {
		if _, err := store.EnsureDevUser(); err != nil {
			log_util.Fatal("初始化DEV用户失败: %v", err)
		}
	}

	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	engine.Use(cors.New(config.Get().Base.Cors.GinCorsConfig()))

	engine.GET("/", h.V(func() string { return "Hello, World!" }))

	// 注册 lab 模块路由（/lab-v1）。
	lab.Bind(engine)

	// 注册 forum 模块路由（/forum-v1，当前为空骨架）。
	forum.Bind(engine)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	monitors.MustInitialize(ctx)

	if !env.DEV {
		gin.SetMode(gin.ReleaseMode)
	}

	handler := labsession.Manager.LoadAndSave(forumsession.Manager.LoadAndSave(engine))

	if env.DEV || !config.Get().Base.TLS.Enabled {
		run(handler, config.Get().Base.Expose, stop)
	} else {
		runTLS(handler, stop)
		run(handler, config.Get().Base.TLS.HttpPort, stop)
	}

	<-ctx.Done()
	log_util.Info("清理中...")

	log_util.Info("清除正在运行的任务")
	var wg sync.WaitGroup
	tasks.RangeExecutors(func(taskId uint, executor *tasks.Executor) {
		wg.Go(func() {
			executor.Interrupt()
			// Done 在任务函数退出、monitor 完成收尾后关闭。
			<-executor.Done()
		})
	})
	wg.Wait()
	log_util.Info("清除完毕")
}

// runCheckConfig 用来在部署前确认配置与二进制的版本匹配
func runCheckConfig() {
	config.MustLoad()
	fmt.Println("配置校验通过")
}

// runGenPolicy 把策略配置与模板渲染成 RAM 策略，结果只写到 stdout。
func runGenPolicy() {
	fs := flag.NewFlagSet("gen_policy", flag.ExitOnError)
	configPath := fs.String("config", policygen.DefaultConfigPath, "策略配置文件路径")
	templatePath := fs.String("template", policygen.DefaultTemplatePath, "策略模板路径")
	envName := fs.String("env", "", "只输出指定环境的纯 JSON；留空输出全部环境")
	_ = fs.Parse(os.Args[2:])

	cfg, err := policygen.Load(*configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "读取策略配置失败: %v\n可从 %s 复制一份并填写真实值\n", err, policygen.ExampleConfigPath)
		} else {
			fmt.Fprintf(os.Stderr, "读取策略配置失败: %v\n", err)
		}
		os.Exit(1)
	}

	out, err := policygen.Render(cfg, *templatePath, *envName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "渲染策略失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(out)
}

func runMigrate() {
	// migrate 只需要数据库配置。
	config.MustLoadStore()
	store.MustInitialize()
	if err := store.Migrate(); err != nil {
		log_util.Fatal("数据库迁移失败: %v", err)
	}
	log_util.Info("数据库迁移完成")
}

func runCreateUser() {
	config.MustLoadStore()
	store.MustInitialize()

	fs := flag.NewFlagSet("create_user", flag.ExitOnError)
	username := fs.String("username", "", "用户名")
	password := fs.String("password", "", "密码")
	role := fs.String("role", "basic", "角色 (basic, operator, superuser)")
	fs.Parse(os.Args[2:])

	if *username == "" || *password == "" {
		fmt.Fprintf(os.Stderr, "错误: --username 和 --password 为必填项\n")
		os.Exit(1)
	}

	validRoles := map[string]perm_util.Role{
		"basic":     perm_util.RoleBasic,
		"operator":  perm_util.RoleOperator,
		"superuser": perm_util.RoleSuperuser,
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

func runTLS(handler http.Handler, cancel context.CancelFunc) {
	go func() {
		tlsCfg := config.Get().Base.TLS
		if err := http.ListenAndServeTLS(":443", tlsCfg.CertFile, tlsCfg.KeyFile, handler); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
			cancel()
		}
	}()
}

func run(handler http.Handler, port int, cancel context.CancelFunc) {
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), handler); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
			cancel()
		}
	}()
}
