package redis

import (
	"fmt"

	"github.com/go-redis/redis"
)

var RedisC *redis.Client

func InitRedis() {
	RedisC = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	_, err := RedisC.Ping().Result()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("#[Redis]: Started Successfully!")
}
