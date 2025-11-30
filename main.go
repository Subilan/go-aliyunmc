package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Subilan/gomc-server/clients"
	"github.com/Subilan/gomc-server/config"
	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/handlers/describe"
	"github.com/Subilan/gomc-server/handlers/ecsActions"
	"github.com/Subilan/gomc-server/helpers"

	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml/v2"
)

func bindRoutes(r *gin.Engine) {
	r.GET("/describe/instanceTypeAndPricePerHour", helpers.QueryHandler(describe.InstanceTypeAndSpotPricePerHour()))
	r.GET("/describe/instance/:instanceId", describe.Instance())
	r.POST("/ecs-actions/createInstance", helpers.BodyHandler(ecsActions.CreateInstance()))
	r.DELETE("/ecs-actions/deleteInstance/:instanceId", ecsActions.DeleteInstance())
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
	log.Print("Loading global Zone information...")

	globals.Zones, err = helpers.RetrieveZones(globals.EcsClient)

	if err != nil {
		log.Fatalln("Error getting zones:", err)
	}

	log.Print("OK")
	log.Print("Loading gin...")

	engine := gin.New()

	bindRoutes(engine)

	err = engine.Run(fmt.Sprintf(":%d", config.Cfg.Server.Expose))

	if err != nil {
		log.Fatalln("cannot start server:", err)
	}
}
