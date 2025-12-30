package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func ConnectRedis() *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		Protocol: 2,
	})
	return redisClient
}

func TestRedis(client *redis.Client) {
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		fmt.Println("Could not connect to redis server")
	} else {
		fmt.Printf("Connected to redis server: %s", pong)
	}
}
