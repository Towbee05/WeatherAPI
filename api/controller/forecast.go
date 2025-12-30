package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Towbee05/weather-service/internal/cache"
	"github.com/Towbee05/weather-service/internal/dto"
	"github.com/Towbee05/weather-service/internal/errors"
	"github.com/gin-gonic/gin"
)

func ForecastController(context *gin.Context) {
	// context.JSON(200, gin.H{"message": "Hello, world"})
	city := context.Query("city")
	if city == "" || city == " " {
		context.JSON(400, gin.H{
			"status":  400,
			"message": "Please provide a city name",
		})
		return
	}
	// Check item in Cache
	var cacheKey = fmt.Sprintf("%sForecast", city)
	itemData, hit := cache.GetItemFromCache(cacheKey)
	var data dto.WeatherResponseForForecast
	if hit {
		json.Unmarshal([]byte(itemData), &data)
		context.JSON(200, data)
		return
	}
	_, found, err := dto.CheckLocationInDB(city)
	if err != nil {
		context.JSON(http.StatusBadRequest, errors.FailOnError(http.StatusBadRequest, "Error while scanning DB for city", err))
	}
	if found == false && err == nil {
		apiResponse, _, _ := dto.FetchForecastDataFromAPI(city)
		_, err := dto.SaveForecastDataToDatabase(apiResponse)
		if err != nil {
			context.JSON(http.StatusInternalServerError, errors.FailOnError(http.StatusInternalServerError, "Error while saving data to DB", err))
			return
		}
		context.JSON(200, apiResponse)
		return
	}
	response := dto.FetchForecastDataFromDB(city)
	cache.SetItemToCache(cacheKey, nil, &response)
	context.JSON(200, response)
}
