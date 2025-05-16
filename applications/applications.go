package applications

import (
	"encoding/json"
	"fmt"
	"slices"

	"bot/utils"

	"bot/datastructs"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
)

var rds *redis.Client

var Applications datastructs.AllApps
var Invoke chan int = make(chan int)
var Output chan error = make(chan error)
var WriteToPending chan datastructs.Application = make(chan datastructs.Application)
var MarkAsReviewed chan datastructs.ApplicationVerdict = make(chan datastructs.ApplicationVerdict)
var FetchApplications chan datastructs.AllApps = make(chan datastructs.AllApps)

func ShowApplication(ds *discordgo.Session, s *datastructs.Session) {
	if len(Applications) > 0 {
		for _, appgroup := range Applications {
			if len(appgroup.PendingApplications) > 0 {
				application := appgroup.PendingApplications[0]
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

				reviewingapp := &datastructs.ReviewingApplication{
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
				break
			}
		}
	}
}

func HandleRedisInput() {
	for {
		id := <-Invoke
		if id == 1 { //Submit Application
			incomingApp := <-WriteToPending
			appgroup, exists := Applications[incomingApp.Author.User.ID]
			if !exists {
				appgroup = datastructs.UserApplicationGroup{
					PendingApplications:  make([]datastructs.Application, 1),
					ReviewedApplications: make([]datastructs.Application, 1),
				}
			}
			appgroup.PendingApplications = append(appgroup.PendingApplications, incomingApp)

			jsonString, err := json.Marshal(appgroup)
			if err != nil {
				fmt.Println("Failed to Marshal")
				Output <- err
				return
			}

			err = rds.HSet("applications", incomingApp.Author.User.ID, jsonString).Err()
			if err != nil {
				fmt.Printf("could not set data in hash: %v", err)
				Output <- err
				return
			}
			Output <- nil
		}
		if id == 2 {
			pointer := <-MarkAsReviewed
			userappgroup := Applications[pointer.App.Author.User.ID]
			for i, app := range userappgroup.PendingApplications {
				if &app == pointer.App {
					userappgroup.PendingApplications = slices.Delete(userappgroup.PendingApplications, i, i)
					pointer.App.Verdict = &pointer.Verdict
					userappgroup.ReviewedApplications = append(userappgroup.ReviewedApplications, *pointer.App)
				}
			}
		}
		if id == 56 {
			fmt.Println("waht")
			FetchApplications <- Applications
		}
	}
}

func OnStart(r *redis.Client) {
	rds = r
	data, err := rds.HGetAll("applications").Result()
	if err != nil {
		fmt.Printf("Could not retrieve all applications from redis.")
		fmt.Println(err.Error())
		return
	}
	Applications = make(datastructs.AllApps)
	for field, value := range data {

		var appgroup datastructs.UserApplicationGroup
		err = json.Unmarshal([]byte(value), &appgroup)
		if err != nil {
			fmt.Println("Failed to sssUnmarshal ", err)
			return
		}
		Applications[field] = appgroup
	}
	go HandleRedisInput()
	fmt.Println("all working up over here")
}
