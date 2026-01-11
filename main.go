package main

import (
	"fmt"
	"log"

	"github.com/Subilan/go-aliyunmc/clients"
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/globals"
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

// TODO: 状态页面，emptyServer测试

func bindRoutes(r *gin.Engine) {
	i := r.Group("/instance")
	ij := i.Group("")
	ij.Use(mid.JWTAuth())
	ia := ij.Group("")
	ia.Use(mid.Role(consts.UserRoleAdmin))

	i.GET("/:instanceId", instances.HandleGetInstance())
	i.GET("", instances.HandleGetActiveOrLatestInstance())
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

	sj := r.Group("/server")
	sj.Use(mid.JWTAuth())
	sj.GET("/exec", server.HandleServerExecute())
	sj.GET("/query", server.HandleServerQuery())
	sj.GET("/info", server.HandleGetServerInfo())
	sj.GET("/backups", server.HandleGetBackupInfo())
	sj.GET("/latest-success-backup", server.HandleGetLatestSuccessBackup())
	sj.GET("/latest-success-archive", server.HandleGetLatestSuccessArchive())

	bj := r.Group("/bss")
	bj.Use(mid.JWTAuth())
	bj.GET("/transactions", bss.HandleGetTransactions())
	bj.GET("/overview", bss.HandleGetOverview())

	r.GET("/ping", simple.HandleGenerate200())
	r.GET("/stream", mid.JWTAuth(), handlers.HandleBeginStream())
}

func runMonitors() {
	var quitActiveInstance = make(chan bool)
	var quitServerStatus = make(chan bool)
	var quitPublicIP = make(chan bool)
	var quitBackup = make(chan bool)
	var quitInstanceCharge = make(chan bool)

	var ip string

	_ = db.Pool.QueryRow("SELECT ip FROM instances WHERE deleted_at IS NULL").Scan(&ip)

	monitors.RestoreInstanceIp(ip)

	go monitors.ActiveInstance(quitActiveInstance)
	go monitors.PublicIP(quitPublicIP)
	go monitors.ServerStatus(quitServerStatus)
	go monitors.Backup(quitBackup)
	go monitors.InstanceCharge(quitInstanceCharge)
}

func main() {
	var err error

	config.Load("config.toml")

	log.Print("Loading global ECS client...")

	globals.EcsClient, err = clients.ShouldCreateEcsClient()

	if err != nil {
		log.Fatalln("Error creating ECS client:", err)
	}

	globals.VpcClient, err = clients.ShouldCreateVpcClient()

	if err != nil {
		log.Fatalln("Error creating VPC client:", err)
	}

	clients.BssClient, err = clients.ShouldCreateBssClient()

	if err != nil {
		log.Fatalln("Error creating BSS client:", err)
	}

	log.Print("Loading global Zone information...")

	globals.ZoneCache, err = globals.RetrieveZones(globals.EcsClient)

	if err != nil {
		log.Fatalln("Error getting zones:", err)
	}

	log.Print("OK")

	log.Print("Loading global VSwitch information...")

	globals.VSwitchCache, err = globals.RetrieveVSwitches(globals.VpcClient)

	if err != nil {
		log.Fatalln("Error getting VSwitchCache:", err)
	}

	log.Print("OK")

	log.Print("Initializing database pool...")

	err = db.InitPool()

	if err != nil {
		log.Fatalln("Error initializing database:", err)
	}

	log.Print("OK")

	log.Println("Loading commands...")

	commands.Load()

	log.Println("Starting monitors...")

	runMonitors()

	log.Print("Loading gin...")

	engine := gin.New()

	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Last-Event-Id"},
		AllowCredentials: true,
	}))

	instances.StartDeployInstanceTaskStatusBroker()

	bindRoutes(engine)

	err = engine.Run(fmt.Sprintf(":%d", config.Cfg.Base.Expose))

	if err != nil {
		log.Fatalln("cannot start server:", err)
	}
}
