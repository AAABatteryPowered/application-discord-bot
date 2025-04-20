package apps

import (
	"encoding/json"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
)

var rds *redis.Client

type Application struct {
	Link    string
	Author  *discordgo.Member
	Verdict *bool
}

type ReviewingApplication struct {
	App     *Application
	EmbedID string
}

type UserApplicationGroup []Application

type allApps map[string]UserApplicationGroup

var Applications allApps
var reviewingApps map[string]Application

func (apps allApps) GetAll() allApps {
	if !(len(reviewingApps) > 0) {
		return nil
	}
	return Applications
}

func OnStart(r *redis.Client) {
	rds = r
	data, err := rds.HGetAll("applications").Result()
	if err != nil {
		fmt.Printf("Could not retrieve all applications from redis.")
		fmt.Println(err.Error())
	}
	Applications = make(map[string]UserApplicationGroup)
	for field, value := range data {
		var appgroup UserApplicationGroup
		err = json.Unmarshal([]byte(value), &appgroup)
		if err != nil {
			fmt.Println("Failed to Unmarshal")
			return
		}
		Applications[field] = appgroup
	}
}
