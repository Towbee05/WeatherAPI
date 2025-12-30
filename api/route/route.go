package route

import (
	"github.com/Towbee05/weather-service/api/controller"
	"github.com/gin-gonic/gin"
)

func CurrentRouter(group *gin.RouterGroup) {
	group.GET("/current", controller.CurrentController)
}

func ForecastRouter(group *gin.RouterGroup) {
	group.GET("/forecast", controller.ForecastController)
}

func Setup(gin *gin.Engine) {
	publicRouter := gin.Group("/api/v1/weather")
	CurrentRouter(publicRouter)
	ForecastRouter(publicRouter)
}
