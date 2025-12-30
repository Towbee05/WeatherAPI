package cache

import (
	_ "github.com/lib/pq"
)

// Function to retrieve item from the cache

func GetItemFromCache(key string) (string, bool) {
	redisClient := ConnectRedis()
	value, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return "An error occured while getting data from cache", false
	} else {
		return value, true
	}
}
