package main

import (
	"fmt"
	"log"

	"github.com/Subilan/go-aliyunmc/clients"
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/events/stream"
	"github.com/Subilan/go-aliyunmc/filelog"
	"github.com/Subilan/go-aliyunmc/handlers"
	"github.com/Subilan/go-aliyunmc/handlers/auth"
	"github.com/Subilan/go-aliyunmc/handlers/bss"
	"github.com/Subilan/go-aliyunmc/handlers/instances"
	"github.com/Subilan/go-aliyunmc/handlers/server"
	"github.com/Subilan/go-aliyunmc/handlers/simple"
	"github.com/Subilan/go-aliyunmc/handlers/tasks"
	"github.com/Subilan/go-aliyunmc/handlers/users"
	"github.com/Subilan/go-aliyunmc/helpers/commands"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/mid"
	"github.com/Subilan/go-aliyunmc/monitors"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func bindRoutes(r *gin.Engine) {
	i := r.Group("/instance")
	ij := i.Group("")
	ij.Use(mid.JWTAuth())
	ia := ij.Group("")
	ia.Use(mid.Role(consts.UserRoleAdmin))

	i.GET("/:instanceId", instances.HandleGetInstance())
	i.GET("", instances.HandleGetInstance())
	i.GET("/active-or-latest", instances.HandleGetActiveOrLatestInstance())
	i.GET("/status", instances.HandleGetActiveInstanceStatus())
	i.GET("/preferred-charge", instances.HandleGetPreferredInstanceCharge())
	i.GET("/desc/:instanceId", instances.HandleDescribeInstance())
	ij.GET("/create-and-deploy", instances.HandleCreateAndDeployInstance())
	ia.GET("/create-preferred", instances.HandleCreatePreferredInstance())
	ia.GET("/deploy", instances.HandleDeployInstance())
	ia.DELETE("/:instanceId", instances.HandleDeleteInstance())
	ia.DELETE("", instances.HandleDeleteInstance())

	u := r.Group("/user")
	uj := u.Group("")
	uj.Use(mid.JWTAuth())
	ua := uj.Group("")
	ua.Use(mid.Role(consts.UserRoleAdmin))

	u.POST("", users.HandleCreateUser())
	uj.GET("", users.HandleGetSelf())
	uj.PATCH("/:userId", users.HandleUserUpdate())
	uj.DELETE("/:userId", users.HandleUserDelete())

	au := r.Group("/auth")
	auj := au.Group("")
	auj.Use(mid.JWTAuth())

	au.POST("/token", auth.HandleGetToken())
	auj.GET("/payload", auth.HandleGetPayload())
	auj.GET("/ping", simple.HandleGenerate200())

	tj := r.Group("/task")
	tj.Use(mid.JWTAuth())
	tj.GET("/s", tasks.HandleGetTasks())
	tj.GET("/overview", tasks.HandleGetTaskOverview())
	tj.GET("/:taskId", tasks.HandleGetTask())
	tj.GET("", tasks.HandleGetActiveTaskByType())
	tj.GET("/cancel/:taskId", tasks.HandleCancelTask())

	s := r.Group("/server")
	s.GET("/info", server.HandleGetServerInfo())
	sj := s.Group("")
	sj.Use(mid.JWTAuth())
	sj.GET("/exec", server.HandleServerExecute())
	sj.GET("/query", server.HandleServerQuery())
	sj.GET("/backups", server.HandleGetBackupInfo())
	sj.GET("/latest-success-backup", server.HandleGetLatestSuccessBackup())
	sj.GET("/latest-success-archive", server.HandleGetLatestSuccessArchive())
	sj.GET("/exec/s", server.HandleGetCommandExecs())
	sj.GET("/exec-overview", server.HandleGetCommandExecOverview())

	bj := r.Group("/bss")
	bj.Use(mid.JWTAuth())
	bj.GET("/transactions", bss.HandleGetTransactions())
	bj.GET("/overview", bss.HandleGetOverview())

	r.GET("/ping", simple.HandleGenerate200())
	r.GET("/", simple.HandleVersion())
	r.GET("/stream", mid.JWTAuth(), handlers.HandleBeginStream())
	r.GET("/stream/simple-public", handlers.HandleBeginSimplePublicStream())
}

func runMonitors() {
	var quitActiveInstance = make(chan bool)
	var quitServerStatus = make(chan bool)
	var quitPublicIP = make(chan bool)
	var quitBackup = make(chan bool)
	var quitInstanceCharge = make(chan bool)
	var quitEmptyServer = make(chan bool)
	var quitBssSync = make(chan bool)

	var ip string

	_ = db.Pool.QueryRow("SELECT ip FROM instances WHERE deleted_at IS NULL").Scan(&ip)

	monitors.RestoreInstanceIp(ip)

	go monitors.ActiveInstance(quitActiveInstance)
	go monitors.PublicIP(quitPublicIP)
	go monitors.ServerStatus(quitServerStatus)
	go monitors.Backup(quitBackup)
	go monitors.InstanceCharge(quitInstanceCharge)
	go monitors.EmptyServer(quitEmptyServer)
	go monitors.BssSync(quitBssSync)
}

// mainLogWriter 是指向 main.log 日志文件的日志 writer
var mainLogWriter = filelog.NewLogWriter("main")

func main() {
	var err error

	log.SetOutput(mainLogWriter)

	config.Load("config.toml")

	log.Print("Loading global ECS client...")

	clients.EcsClient, err = clients.ShouldCreateEcsClient()

	if err != nil {
		log.Fatalln("Error creating ECS client:", err)
	}

	clients.VpcClient, err = clients.ShouldCreateVpcClient()

	if err != nil {
		log.Fatalln("Error creating VPC client:", err)
	}

	clients.BssClient, err = clients.ShouldCreateBssClient()

	if err != nil {
		log.Fatalln("Error creating BSS client:", err)
	}

	log.Print("Initializing database pool...")

	err = db.InitPool()

	if err != nil {
		log.Fatalln("Error initializing database:", err)
	}

	log.Println("Loading commands...")

	commands.Load()

	log.Println("Starting monitors...")

	runMonitors()

	log.Println("Intializing global stream...")

	stream.InitPublicChannel()

	log.Print("Loading gin...")

	engine := gin.New()

	gin.DefaultWriter = mainLogWriter

	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	engine.Use(cors.New(config.Cfg.Base.GetGinCorsConfig()))

	instances.StartDeployInstanceTaskStatusBroker()

	bindRoutes(engine)

	err = engine.Run(fmt.Sprintf("0.0.0.0:%d", config.Cfg.Base.Expose))

	if err != nil {
		log.Fatalln("cannot start server:", err)
	}
}
