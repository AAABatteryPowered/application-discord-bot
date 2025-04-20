package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
	. "midas.com/bot/applications"
	"midas.com/bot/sessions"
	"midas.com/bot/utils"
)

// Bot Token (You need to replace this with your bot's token)
const token = "MTMyNzU4Mjg0MTkwNjc5NDU1Ng.GsPPqZ.YSbhP_b0wcJ-PHRmOXTl9PNwySJWrOAC2kFmxI"
const guildid = "1355623019971608706"

var rds *redis.Client

//var reviewedApplications map[int]UserApplicationGroup

var roleAppReviewer string = "1358026605330563185"

// This function will be called when the bot is ready
func ready(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println("Bot is now online!")
	sessions.Start(s)
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
		Link:    link,
		Author:  user,
		Verdict: nil,
	}

	appgroup, exists := Applications[user.User.ID]
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

func serveApplication(s *discordgo.Session, ss *sessions.Session) {
	apps := Applications.GetAll()
	if apps != nil {
		for _, appgroup := range Applications {
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

				msg, err := s.ChannelMessageSendEmbed(ss.SessionChannel, embed)
				if err != nil {
					fmt.Println("error sending embed:", err)
					return
				}

				//currentlyReviewingApp = application
				//currentlyReviewingAppMsg = msg

				// React to the message with a ✅ emoji
				err = s.MessageReactionAdd(ss.SessionChannel, msg.ID, "✅")
				if err != nil {
					fmt.Println("error adding reaction:", err)
					return
				}

				err = s.MessageReactionAdd(ss.SessionChannel, msg.ID, "❌")
				if err != nil {
					fmt.Println("error adding reaction:", err)
					return
				}
			}
			break
		}
	}
}

func broadcastApplicationDecision(s *discordgo.Session, accepted bool, app Application) {
	channel, err := s.UserChannelCreate(app.Author.User.ID)
	if err != nil {
		fmt.Println("Error creating DM channel:", err)
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Your application has been denied ❌",
		Description: "Unfortunately, the application reviewing team has decided to **deny you** from the Midas SMP. Your best bet to get accepted is to reread the application rules/standards, learn some new skills and submit another new and improved application.",
		Color:       0xff4070, // teal-ish
		Footer: &discordgo.MessageEmbedFooter{
			Text: "- Midas SMP Application Reviewers",
		},
	}

	// Step 2: Send the message
	_, err = s.ChannelMessageSendEmbed(channel.ID, embed)
	if err != nil {
		fmt.Println("Error sending Embed DM:", err)
		return
	}
}

var botCategory string = "1359830076455125185"

func ReactionHandler(s *discordgo.Session, i *discordgo.MessageReactionAdd) {
	/*msg, err := s.ChannelMessage(i.ChannelID, i.MessageID)
	if err != nil {
		fmt.Println("Error fetching message:", err)
		return
	}
	if msg.Author.ID == s.State.User.ID && i.MessageID == currentlyReviewingAppMsg.ID && i.UserID != s.State.User.ID {
		if i.Emoji.Name == "✅" {
			broadcastApplicationDecision(s, true, currentlyReviewingApp)
		} else if i.Emoji.Name == "❌" {
			broadcastApplicationDecision(s, false, currentlyReviewingApp)
		}
	}*/
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
			sessions.SessionOpen(s, i)
			/*channelExists, err := utils.GetChannelInCategoryByName(s, guildid, botCategory, fmt.Sprintf("session-%s", i.Member.User.Username))
			if err != nil {
				fmt.Println(err)
				return
			}
			if channelExists != nil {

			}
			reviewApplicationCycle(s, i)
			reviewingchan := make(chan int, 1)
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
	OnStart(rds)
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

		sessions.Shutdown()

		dg.Close()
		s.Close()
	}(dg)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
