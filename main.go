package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/remiges-aniket/configsvc"
	"github.com/remiges-aniket/etcd"
	"github.com/remiges-aniket/rigel"
	"github.com/remiges-aniket/utils"
	"github.com/remiges-tech/alya/config"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/logharbour"
)

const dialTimeout = 5 * time.Second

func main() {

	appConfig, environment := setConfigEnvironment(utils.DevEnv)

	// logger
	// Open a file for logging.
	logFile, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	// Create a fallback writer that uses the file as the primary writer and stdout as the fallback.
	fallbackWriter := logharbour.NewFallbackWriter(logFile, os.Stdout)
	lctx := logharbour.NewLoggerContext(logharbour.Debug0)
	lh := logharbour.NewLogger(lctx, "Rigel", fallbackWriter)
	lh.WithPriority(logharbour.Info)

	// Rigel
	// error types
	// Open the error types file
	file, err := os.Open("./errortypes.yaml")
	if err != nil {
		log.Fatalf("Failed to open error types file: %v", err)
	}
	defer file.Close()
	// Load the error types
	wscutils.LoadErrorTypes(file)
	// Router
	r := gin.Default()
	if environment == utils.DevEnv {
		r.Use(corsMiddleware())
	}
	// Database connection
	cli, err := etcd.NewEtcdStorage([]string{fmt.Sprint(appConfig.DBHost + ":" + strconv.Itoa(appConfig.DBPort))})

	if err != nil {
		fmt.Print("error", err)
		return
	}

	rigel.NewWithStorage(cli)

	// Services
	// Config Services
	s := service.NewService(r).WithDependency("client", cli).WithLogHarbour(lh).WithDependency("appConfig", appConfig)
	s.RegisterRoute(http.MethodGet, "/configget", configsvc.Config_get)
	s.RegisterRoute(http.MethodGet, "/configlist", configsvc.Config_list)
	s.RegisterRoute(http.MethodPost, "/configset", configsvc.Config_set)

	r.Run(":" + appConfig.AppServerPort)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}

func setConfigEnvironment(environment utils.Environment) (utils.AppConfig, utils.Environment) {
	var appConfig utils.AppConfig
	if !environment.IsValid() {
		log.Fatal("environment params is not valid")
	}
	switch environment {
	case utils.DevEnv:
		config.LoadConfigFromFile("./config_dev.json", &appConfig)
	case utils.ProdEnv:
		config.LoadConfigFromFile("./config_prod.json", &appConfig)
	case utils.UATEnv:
		config.LoadConfigFromFile("./config_uat.json", &appConfig)
	}
	return appConfig, environment
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
