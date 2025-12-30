package controller

import (
	"encoding/json"
	"net/http"

	"github.com/Towbee05/weather-service/internal/cache"
	"github.com/Towbee05/weather-service/internal/dto"
	"github.com/Towbee05/weather-service/internal/errors"
	"github.com/gin-gonic/gin"
)

func CurrentController(context *gin.Context) {
	city := context.Query("city")
	if city == "" || city == " " {
		context.JSON(400, gin.H{"status": 404, "message": "Could not find specified location"})
		return
	}
	// Check item in cache first
	itemData, hit := cache.GetItemFromCache(city)
	var data dto.WeatherResponseForCurrent
	if hit {
		json.Unmarshal([]byte(itemData), &data)
		context.JSON(200, data)
		return
	}
	_, found, err := dto.CheckLocationInDB(city)
	if err != nil {
		context.JSON(500, errors.FailOnError(http.StatusInternalServerError, "Error while checking city location", err))
	}
	if found == false && err == nil {
		apiResponse, _, _ := dto.FetchForecastDataFromAPI(city)
		_, err := dto.SaveForecastDataToDatabase(apiResponse)
		if err != nil {
			context.JSON(500, errors.FailOnError(http.StatusInternalServerError, "Error While saving data to DB", err))
			return
		}
		data := dto.WeatherResponseForCurrent{
			Location: apiResponse.Location,
			Current:  apiResponse.Current,
		}
		cache.SetItemToCache(city, &data, nil)
		context.JSON(200, data)
		return
	}
	response := dto.FetchCurrentDataFromDB(city)
	cache.SetItemToCache(city, &response, nil)
	context.JSON(200, response)
}
