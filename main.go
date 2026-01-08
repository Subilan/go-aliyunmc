package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Subilan/go-aliyunmc/clients"
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/handlers"
	"github.com/Subilan/go-aliyunmc/handlers/auth"
	"github.com/Subilan/go-aliyunmc/handlers/instances"
	"github.com/Subilan/go-aliyunmc/handlers/server"
	"github.com/Subilan/go-aliyunmc/handlers/simple"
	"github.com/Subilan/go-aliyunmc/handlers/tasks"
	"github.com/Subilan/go-aliyunmc/handlers/users"
	"github.com/Subilan/go-aliyunmc/helpers/commands"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/middlewares"
	"github.com/Subilan/go-aliyunmc/monitors"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml/v2"
)

func bindRoutes(r *gin.Engine) {
	r.GET("/instance/:instanceId", instances.HandleGetInstance())
	r.GET("/active-or-latest-instance", instances.HandleGetActiveOrLatestInstance())
	r.GET("/instance-status", instances.HandleGetActiveInstanceStatus())
	r.GET("/instance-types-and-charge", instances.HandleGetInstanceTypesAndSpotPricePerHour())
	r.GET("/instance-description/:instanceId", instances.HandleDescribeInstance())
	r.POST("/instance", middlewares.JWTAuth(), instances.HandleCreateInstance())
	r.DELETE("/instance/:instanceId", middlewares.JWTAuth(), instances.HandleDeleteInstance())
	r.DELETE("/instance", middlewares.JWTAuth(), instances.HandleDeleteInstance())
	r.GET("/instance-deploy", middlewares.JWTAuth(), instances.HandleDeployInstance())
	r.POST("/user", users.HandleCreateUser())
	r.PATCH("/user/:userId", middlewares.JWTAuth(), users.HandleUserUpdate())
	r.DELETE("/user/:userId", middlewares.JWTAuth(), users.HandleUserDelete())
	r.POST("/auth/token", auth.HandleGetToken())
	r.GET("/auth/payload", middlewares.JWTAuth(), auth.HandleGetPayload())
	r.GET("/authed-ping", middlewares.JWTAuth(), simple.HandleGenerate200())
	r.GET("/ping", simple.HandleGenerate200())
	r.GET("/stream", middlewares.JWTAuth(), handlers.HandleBeginStream())
	r.GET("/task/:taskId", middlewares.JWTAuth(), tasks.HandleGetTask())
	r.GET("/task", middlewares.JWTAuth(), tasks.HandleGetActiveTaskByType())
	r.GET("/task-cancel/:taskId", middlewares.JWTAuth(), tasks.HandleCancelTask())
	r.GET("/server/exec", middlewares.JWTAuth(), server.HandleServerExecute())
	r.GET("/server/query", middlewares.JWTAuth(), server.HandleServerQuery())
	r.GET("/server/info", middlewares.JWTAuth(), server.HandleGetServerInfo())
	r.GET("/server/backups", middlewares.JWTAuth(), server.HandleGetBackupInfo())
	r.GET("/server/latest-success-backup", middlewares.JWTAuth(), server.HandleGetLatestSuccessBackup())
	r.GET("/server/latest-success-archive", middlewares.JWTAuth(), server.HandleGetLatestSuccessArchive())
}

func runMonitors() {
	var quitActiveInstance = make(chan bool)
	var quitServerStatus = make(chan bool)
	var quitPublicIP = make(chan bool)
	var quitBackup = make(chan bool)

	var ip string

	_ = db.Pool.QueryRow("SELECT ip FROM instances WHERE deleted_at IS NULL").Scan(&ip)

	go monitors.ActiveInstance(quitActiveInstance)
	go monitors.PublicIP(quitPublicIP)
	go monitors.ServerStatus(quitServerStatus)
	go monitors.Backup(quitBackup)
}

func main() {
	log.Print("Loading config...")

	configFileContent, err := os.ReadFile("config.toml")

	if err != nil {
		log.Fatalln("Error reading config file:", err)
	}

	err = toml.Unmarshal(configFileContent, &config.Cfg)

	if err != nil {
		log.Fatalln("cannot unmarshal config.toml:", err)
	}

	log.Print("OK")

	log.Print("Loading global ECS client...")

	globals.EcsClient, err = clients.ShouldCreateEcsClient()

	if err != nil {
		log.Fatalln("Error creating ECS client:", err)
	}

	log.Print("OK")

	globals.VpcClient, err = clients.ShouldCreateVpcClient()
	if err != nil {
		log.Fatalln("Error creating VPC client:", err)
	}

	log.Print("OK")

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

	bindRoutes(engine)

	err = engine.Run(fmt.Sprintf(":%d", config.Cfg.Server.Expose))

	if err != nil {
		log.Fatalln("cannot start server:", err)
	}
}
