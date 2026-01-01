package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Subilan/gomc-server/clients"
	"github.com/Subilan/gomc-server/config"
	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/handlers"
	"github.com/Subilan/gomc-server/handlers/auth"
	"github.com/Subilan/gomc-server/handlers/describe"
	"github.com/Subilan/gomc-server/handlers/ecsActions"
	"github.com/Subilan/gomc-server/handlers/get"
	"github.com/Subilan/gomc-server/handlers/server"
	"github.com/Subilan/gomc-server/handlers/simple"
	"github.com/Subilan/gomc-server/handlers/tasks"
	"github.com/Subilan/gomc-server/handlers/users"
	"github.com/Subilan/gomc-server/middlewares"
	"github.com/Subilan/gomc-server/monitors"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml/v2"
)

func bindRoutes(r *gin.Engine) {
	r.GET("/get/instance/:instanceId", get.Instance())
	r.GET("/get/instance-status/:instanceId", get.InstanceStatus())
	r.GET("/describe/instance-types-and-charge", describe.InstanceTypesAndSpotPricePerHour())
	r.GET("/describe/instance/:instanceId", describe.Instance())
	r.POST("/ecs-actions/create-instance", ecsActions.CreateInstance())
	r.DELETE("/ecs-actions/delete-instance/:instanceId", ecsActions.DeleteInstance())
	r.DELETE("/ecs-actions/delete-instance", ecsActions.DeleteInstance())
	r.GET("/ecs-actions/deploy-active-instance", middlewares.JWTAuth(), ecsActions.DeployInstance())
	r.POST("/user/create", users.Create())
	r.PATCH("/user/:userId", middlewares.JWTAuth(), users.Update())
	r.DELETE("/user/:userId", middlewares.JWTAuth(), users.Delete())
	r.POST("/auth/get-token", auth.GetToken())
	r.GET("/auth/get-payload", middlewares.JWTAuth(), auth.GetPayload())
	r.GET("/auth/ping", middlewares.JWTAuth(), simple.Gen200())
	r.GET("/ping", simple.Gen200())
	r.GET("/stream", middlewares.JWTAuth(), handlers.BeginStream())
	r.GET("/task/cancel/:taskId", middlewares.JWTAuth(), tasks.Cancel())
	r.GET("/server/start", middlewares.JWTAuth(), server.Start())
}

func runMonitors() {
	var activeInstanceStatusMonitorErrChan = make(chan error)
	monitors.GlobalActiveInstanceStatusMonitor = monitors.NewActiveInstanceStatusMonitor(context.Background(), activeInstanceStatusMonitorErrChan)
	monitors.GlobalActiveInstanceStatusMonitor.Run()

	var automaticPublicIpAllocatorErrChan = make(chan error)
	monitors.GlobalAutomaticPublicIPAllocator = monitors.NewAutomaticPublicIPAllocator(context.Background(), automaticPublicIpAllocatorErrChan)
	monitors.GlobalAutomaticPublicIPAllocator.Run()
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

	err = globals.InitPool()

	if err != nil {
		log.Fatalln("Error initializing database:", err)
	}

	log.Print("OK")

	runMonitors()

	log.Print("Loading gin...")

	engine := gin.New()

	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	bindRoutes(engine)

	err = engine.Run(fmt.Sprintf(":%d", config.Cfg.Server.Expose))

	if err != nil {
		log.Fatalln("cannot start server:", err)
	}
}
