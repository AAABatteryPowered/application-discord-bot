package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
)

// Bot Token (You need to replace this with your bot's token)
const token = "MTMyNzU4Mjg0MTkwNjc5NDU1Ng.GsPPqZ.YSbhP_b0wcJ-PHRmOXTl9PNwySJWrOAC2kFmxI"
const guildid = "1355623019971608706"

var rds *redis.Client

type Application struct {
	Link   string
	Author *discordgo.User
	ID     string
}

type UserApplicationGroup []Application

var applications map[int]UserApplicationGroup

// This function will be called when the bot is ready
func ready(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println("Bot is now online!")
}

func RegisterCommands(s *discordgo.Session) {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "submit",
			Description: "Submits your application.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "link",
					Description: "The link to your application",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
	}

	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildid, commands)
	if err != nil {
		fmt.Printf("Failed to register commands.")
	} else {
		fmt.Println("Commands registered successfully.")
	}
}

func submitApplication(interaction *discordgo.InteractionCreate, link string) error {
	app := Application{
		Link:   link,
		Author: interaction.User,
		ID:     interaction.User.ID,
	}

	id, err := strconv.Atoi(interaction.User.ID)
	if err != nil {
		fmt.Println("Failed to convert UserID string to integer")
		return err
	}

	appgroup, exists := applications[id]
	if !exists {
		appgroup = make(UserApplicationGroup, 1, 1)
	}
	appgroup = append(appgroup, app)

	jsonString, err := json.Marshal(appgroup)
	if err != nil {
		fmt.Println("Failed to Marshal")
		return err
	}

	err = rds.HSet("applications", interaction.User.ID, jsonString).Err()
	if err != nil {
		fmt.Println("could not set data in hash: %v", err)
		return err
	}
	//discordgo.user is a pointer remember that
	return nil
}

func CommandsHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "submit":

			var message string
			if len(i.ApplicationCommandData().Options) > 0 {
				// If the user has passed an argument for echo
				message = i.ApplicationCommandData().Options[0].StringValue()
			}

			if strings.Contains(message, "https://www.youtube.com") || strings.Contains(message, "https://www.youtu.be") {

				embed := discordgo.MessageEmbed{
					Title:       "rizz",
					Description: "lord",
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{&embed},
					},
				})
			} else {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Sorry, it seems there is no youtube link here! Try submitting again, but this time with a link to your application.",
					},
				})
			}
		default:
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Sorry, I couldn't recognize this command!",
				},
			})
		}
	}
}

func InitRedis() {
	rds = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // No password set
		DB:       0,  // Use default DB
	})
	_, err := rds.Ping().Result()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("Redis started successfully!")
}

func main() {
	// Create a new Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session,", err)
		return
	}

	InitRedis()

	intents := discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuildMembers

	dg.Identify.Intents = intents

	dg.AddHandler(ready)
	dg.AddHandlerOnce(func(s *discordgo.Session, r *discordgo.Ready) {
		RegisterCommands(s)
	})
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		go CommandsHandler(s, i)
	})

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection,", err)
		return
	}
	fmt.Println("Bot is now running. Press CTRL+C to exit.")

	select {}
}
