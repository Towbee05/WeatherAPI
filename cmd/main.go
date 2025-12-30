package main

// TODO 1. Construct structs
// TODO 2. Check db for city
// TODO 3. If city is not in db or error was returned while searching db jump to step 5
// TODO 4. Return city from API
// TODO 5. Ask Chat if I need to run "defer db.Close()" on opening connection to db

import (
	"fmt"

	route "github.com/Towbee05/weather-service/api/route"
	"github.com/Towbee05/weather-service/internal/cache"
	"github.com/Towbee05/weather-service/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

// var ctx = context.Background()

func main() {
	db.ConnectDB()
	// Connect to redis cache for in-memory storage

	var gin *gin.Engine = gin.Default()
	err := godotenv.Load()
	if err != nil {
		fmt.Println("could not access .env files")
		return
	}
	// Engine is running now, check if the redis server is reciving requests

	// Forecast URL
	route.Setup(gin)
	redisCache := cache.ConnectRedis()
	cache.TestRedis(redisCache)

	gin.Run(":8080")
}
