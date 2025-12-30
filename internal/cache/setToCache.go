package cache

import (
	"encoding/json"
	"time"

	"github.com/Towbee05/weather-service/internal/dto"
	_ "github.com/lib/pq"
	// "github.com/redis/go-redis/v9"
)

func SetItemToCache(key string, current *dto.WeatherResponseForCurrent, forecast *dto.WeatherResponseForForecast) string {
	redisClient := ConnectRedis()
	var marshalledData any
	if current != nil {
		marshalledData, _ = json.Marshal(current)
	}
	if forecast != nil {
		marshalledData, _ = json.Marshal(forecast)
	}
	err := redisClient.Set(ctx, key, marshalledData, 15*time.Minute)
	if err != nil {
		return "An error occured while setting data to cache"
	} else {
		return "Item set to cache"
	}
}
