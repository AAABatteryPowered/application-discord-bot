package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
	"midas.com/bot/utils"
)

// Bot Token (You need to replace this with your bot's token)
const token = "MTMyNzU4Mjg0MTkwNjc5NDU1Ng.GsPPqZ.YSbhP_b0wcJ-PHRmOXTl9PNwySJWrOAC2kFmxI"
const guildid = "1355623019971608706"

var rds *redis.Client

type Application struct {
	Link   string
	Author *discordgo.Member
}

type UserApplicationGroup []Application

var applications map[int]UserApplicationGroup

var currentlyReviewingApp Application
var currentlyReviewingAppMsg *discordgo.Message

//var reviewedApplications map[int]UserApplicationGroup

var roleAppReviewer string = "1358026605330563185"

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
		{
			Name:        "review",
			Description: "Enters you into application reviewing mode. Administrators only",
		},
	}

	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildid, commands)
	if err != nil {
		fmt.Printf("Failed to register commands.")
	} else {
		fmt.Println("Commands registered successfully.")
	}
}

func submitApplication(user *discordgo.Member, link string) error {
	app := Application{
		Link:   link,
		Author: user,
	}

	id, err := strconv.Atoi(user.User.ID)
	if err != nil {
		fmt.Println("Failed to convert UserID string to integer")
		return err
	}

	appgroup, exists := applications[id]
	if !exists {
		appgroup = make(UserApplicationGroup, 1)
	}
	appgroup = append(appgroup, app)

	jsonString, err := json.Marshal(appgroup)
	if err != nil {
		fmt.Println("Failed to Marshal")
		return err
	}

	err = rds.HSet("applications", user.User.ID, jsonString).Err()
	if err != nil {
		fmt.Printf("could not set data in hash: %v", err)
		return err
	}
	//discordgo.user is a pointer remember that
	return nil
}

func serveApplication(s *discordgo.Session, c *discordgo.Channel) {
	if len(applications) > 0 {
		for _, appgroup := range applications {
			if len(appgroup) > 0 {
				application := appgroup[1]
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

				msg, err := s.ChannelMessageSendEmbed(c.ID, embed)
				if err != nil {
					fmt.Println("error sending embed:", err)
					return
				}

				// React to the message with a ✅ emoji
				err = s.MessageReactionAdd(c.ID, msg.ID, "✅")
				if err != nil {
					fmt.Println("error adding reaction:", err)
					return
				}

				err = s.MessageReactionAdd(c.ID, msg.ID, "❌")
				if err != nil {
					fmt.Println("error adding reaction:", err)
					return
				}
			}
			break
		}
	}
}

func broadcastApplicationDecision(accepted bool, app Application) {
	fmt.Println("huzz")
}

var botCategory string = "1359830076455125185"
var reviewSessionChannels []string

func reviewApplicationCycle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	botUser, err := s.User("@me")
	if err != nil {
		fmt.Println("error fetching bot user:", err)
		return
	}

	denyAll := discordgo.PermissionOverwrite{
		ID:   guildid,
		Type: discordgo.PermissionOverwriteTypeRole,
		Deny: discordgo.PermissionViewChannel,
	}

	allowUser := discordgo.PermissionOverwrite{
		ID:    i.Member.User.ID,
		Type:  discordgo.PermissionOverwriteTypeMember,
		Allow: discordgo.PermissionViewChannel,
	}

	allowBot := discordgo.PermissionOverwrite{
		ID:    botUser.ID,
		Type:  discordgo.PermissionOverwriteTypeMember,
		Allow: discordgo.PermissionViewChannel,
	}

	channel, err := s.GuildChannelCreateComplex(guildid, discordgo.GuildChannelCreateData{
		Name:     fmt.Sprintf("session-%s", i.Member.User.Username),
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: botCategory,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			&denyAll,
			&allowUser,
			&allowBot,
		},
	})
	if err != nil {
		fmt.Println("error creating channel:", err)
		return
	}
	reviewSessionChannels = append(reviewSessionChannels, channel.ID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Reviewing session created successfully in <#%s>", channel.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	serveApplication(s, channel)
}

func ReactionHandler(s *discordgo.Session, i *discordgo.MessageReactionAdd) {
	fmt.Println("yo")
	msg, err := s.ChannelMessage(i.ChannelID, i.MessageID)
	if err != nil {
		fmt.Println("Error fetching message:", err)
		return
	}

	if msg.Member.User.ID == s.State.User.ID && msg.ID == currentlyReviewingAppMsg.ID && msg.Member.User.ID != s.State.User.ID {
		if i.Emoji.Name == "✅" {
			broadcastApplicationDecision(true, currentlyReviewingApp)
		} else if i.Emoji.Name == "❌" {
			broadcastApplicationDecision(false, currentlyReviewingApp)
		}
	}
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

			if strings.Contains(message, "https://www.youtube.com") || strings.Contains(message, "https://www.youtu.be") || strings.Contains(message, "https://youtu.be") || strings.Contains(message, "https://youtube.com") {
				if i.Member != nil {
					err := submitApplication(i.Member, message)
					var embed discordgo.MessageEmbed

					videoID := utils.ExtractYouTubeID(message)

					thumbnailURL := fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", videoID)

					if err != nil {
						embed = discordgo.MessageEmbed{
							Title:       "Failed to submit your application",
							Description: "Sorry, but an error occurred while submitting your application. Please try again, and if the error persists, you can report it here.",
							Color:       0xff5500,
						}
					} else {
						embed = discordgo.MessageEmbed{
							Title:       "Success",
							Description: "Your application has been submitted and is waiting to be reviewed!",
							Color:       0x00ff7b,
							Thumbnail: &discordgo.MessageEmbedThumbnail{
								URL: thumbnailURL,
							},
						}
						//discordgo.MessageEmbed.Color
					}

					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds: []*discordgo.MessageEmbed{&embed},
							Flags:  discordgo.MessageFlagsEphemeral,
						},
					})
				} else {
					fmt.Println("No user detected!")
				}
			} else {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Sorry, it seems there is no youtube link here! Try submitting again, but this time with a link to your application.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
			}
		case "review":
			hasRole := false
			for _, roleID := range i.Member.Roles {
				if roleID == roleAppReviewer {
					hasRole = true
					break
				}
			}
			if !hasRole {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You do not have permission to do this!",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			channelExists, err := utils.GetChannelInCategoryByName(s, guildid, botCategory, fmt.Sprintf("session-%s", i.Member.User.Username))
			if err != nil {
				fmt.Println(err)
				return
			}
			if channelExists != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("You already have a reviewing session in <#%s>", channelExists.ID),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			reviewApplicationCycle(s, i)
			/*reviewingchan := make(chan int, 1)
			for {
				go serveApplication(s, i, reviewingchan)
				code := <-reviewingchan
				if code == 1 {
					go broadcastApplicationDecision()
				}
			}*/

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
	// Restore Data
	data, err := rds.HGetAll("applications").Result()
	if err != nil {
		fmt.Printf("Could not retrieve all applications from redis.")
		fmt.Println(err.Error())
	}
	applications = make(map[int]UserApplicationGroup)
	for field, value := range data {
		key, err := strconv.Atoi(field)
		if err != nil {
			fmt.Println("Failed to convert hash key string to integer.")
			return
		}
		var appgroup UserApplicationGroup
		err = json.Unmarshal([]byte(value), &appgroup)
		if err != nil {
			fmt.Println("Failed to Unmarshal")
			return
		}
		applications[key] = appgroup
	}
}

func main() {
	// Create a new Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session,", err)
		return
	}

	InitRedis()

	intents := discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessageReactions

	dg.Identify.Intents = intents

	dg.AddHandler(ready)
	dg.AddHandlerOnce(func(s *discordgo.Session, r *discordgo.Ready) {
		RegisterCommands(s)
	})
	dg.AddHandler(CommandsHandler)
	dg.AddHandler(ReactionHandler)
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection,", err)
		return
	}
	fmt.Println("Bot is now running. Press CTRL+C to exit.")

	// Shutdown

	defer func(s *discordgo.Session) {
		fmt.Println("Running shutdown logic...")
		for _, id := range reviewSessionChannels {
			s.ChannelDelete(id)
		}

		dg.Close()
		s.Close()
	}(dg)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
