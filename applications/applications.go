package apps

import (
	"encoding/json"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
	. "midas.com/bot/datastructs"
	"midas.com/bot/utils"
)

var rds *redis.Client

func ShowApplication(ds *discordgo.Session, s *Session) {
	if len(Applications) > 0 {
		for _, appgroup := range Applications {
			if len(appgroup) > 0 {
				application := appgroup[0]
				videoID := utils.ExtractYouTubeID(application.Link)
				thumbnailURL := fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", videoID)

				embed := &discordgo.MessageEmbed{
					Title:       application.Author.User.Username,
					Description: application.Link,
					Color:       0x00ff7b,
					Image: &discordgo.MessageEmbedImage{
						URL: thumbnailURL,
					},
				}

				msg, err := ds.ChannelMessageSendEmbed(s.SessionChannel, embed)
				if err != nil {
					fmt.Println("error sending embed:", err)
					return
				}

				reviewingapp := &ReviewingApplication{
					App:     &application,
					EmbedID: msg.ID,
				}

				s.CurrentApp = reviewingapp

				err = ds.MessageReactionAdd(s.SessionChannel, msg.ID, "✅")
				if err != nil {
					fmt.Println("error adding reaction:", err)
					return
				}

				err = ds.MessageReactionAdd(s.SessionChannel, msg.ID, "❌")
				if err != nil {
					fmt.Println("error adding reaction:", err)
					return
				}
			}
			break
		}
	}
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

	data, err = rds.HGetAll("pastapplications").Result()
	if err != nil {
		fmt.Printf("Could not retrieve all past applications from redis.")
		fmt.Println(err.Error())
	}
	PastApplications = make(map[string]UserApplicationGroup)
	for field, value := range data {
		var appgroup UserApplicationGroup
		err = json.Unmarshal([]byte(value), &appgroup)
		if err != nil {
			fmt.Println("Failed to Unmarshal")
			return
		}
		PastApplications[field] = appgroup
	}
}
