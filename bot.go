package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Bot Token (You need to replace this with your bot's token)
const token = "MTMyNzU4Mjg0MTkwNjc5NDU1Ng.GsPPqZ.YSbhP_b0wcJ-PHRmOXTl9PNwySJWrOAC2kFmxI"
const guildid = "1355623019971608706"

// This function will be called when the bot is ready
func ready(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println("Bot is now online!")
}

func RegisterCommands(s *discordgo.Session) {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "submit",
			Description: "Submits your application.",
		},
	}

	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildid, commands)
	if err != nil {
		fmt.Printf("Failed to register commands.")
	} else {
		fmt.Println("Commands registered successfully.")
	}
}

func CommandsHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "submit":

			message := "12"
			if len(i.ApplicationCommandData().Options) > 0 {
				// If the user has passed an argument for echo
				message = i.ApplicationCommandData().Options[0].StringValue()
			}

			fmt.Println(message)
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

func main() {
	// Create a new Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session,", err)
		return
	}

	intents := discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuildMembers

	dg.Identify.Intents = intents

	dg.AddHandler(ready)
	dg.AddHandler(RegisterCommands)
	dg.AddHandler(CommandsHandler)

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection,", err)
		return
	}
	fmt.Println("Bot is now running. Press CTRL+C to exit.")

	select {}
}
