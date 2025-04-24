package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
	. "midas.com/bot/applications"
	. "midas.com/bot/datastructs"
	"midas.com/bot/sessions"
	"midas.com/bot/utils"
)

// Bot Token (You need to replace this with your bot's token)
const token = "MTMyNzU4Mjg0MTkwNjc5NDU1Ng.GsPPqZ.YSbhP_b0wcJ-PHRmOXTl9PNwySJWrOAC2kFmxI"
const guildid = "1355623019971608706"

var rds *redis.Client

var roleAppReviewer string = "1358026605330563185"

func ready(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println("Bot is now online!")
	sessions.Start(s)
	RegisterCommands(s)
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
		{
			Name:        "listapps",
			Description: "Lists every pending application. Administrators only",
		},
		{
			Name:        "clearapps",
			Description: "Deletes All Applications. Administrators only",
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
	appgroup, exists := Applications[user.User.ID]
	if !exists {
		appgroup = make(UserApplicationGroup, 1)
	}

	app := Application{
		Link:    link,
		Author:  user,
		Verdict: nil,
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

	return nil
}

func handleApplicationDecision(s *discordgo.Session, accepted bool, app Application) {
	channel, err := s.UserChannelCreate(app.Author.User.ID)
	if err != nil {
		fmt.Println("Error creating DM channel:", err)
		return
	}

	var embed *discordgo.MessageEmbed

	if accepted {
		invite, err := s.ChannelInviteCreate("1343336810734026835", discordgo.Invite{
			MaxUses:   1,
			MaxAge:    0,
			Temporary: false,
			Unique:    true,
		})
		if err != nil {
			fmt.Println(err)
		}
		embed = &discordgo.MessageEmbed{
			Title:       "Congrats! Your application has been accepted ✅",
			Description: fmt.Sprintf("Great job on your application, as the reviewing team have decided you seem like a good fit for the server! A one time invite is attached below, and it does have verification so don't try and invite anyone else. Good luck on the server! https://discord.gg/" + invite.Code),
			Color:       0xBAFF29,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "- Midas SMP Application Reviewers",
			},
		}
	} else {
		embed = &discordgo.MessageEmbed{
			Title:       "Your application has been denied ❌",
			Description: "Unfortunately, the application reviewing team has decided to **deny you** from the Midas SMP. Your best bet to get accepted is to reread the application rules/standards, learn some new skills and submit another new and improved application.",
			Color:       0xff4070,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "- Midas SMP Application Reviewers",
			},
		}
	}

	var decidedApps UserApplicationGroup = make(UserApplicationGroup, 1)

	fmt.Println("pluh", app)
	for i, v := range Applications[app.Author.User.ID] {
		if v == app {
			fmt.Println(Applications[app.Author.User.ID])
			Applications[app.Author.User.ID] = slices.Delete(Applications[app.Author.User.ID], i, i)
			fmt.Println(Applications[app.Author.User.ID])
		}
	}

	marshalled, err := json.Marshal(Applications[app.Author.User.ID])
	if err != nil {
		return
	}
	err = rds.HSet("applications", app.Author.User.ID, marshalled).Err()
	if err != nil {
		return
	}
	marshalled, err = json.Marshal(decidedApps)
	if err != nil {
		return
	}
	err = rds.HSet("reviewedapplications", app.Author.User.ID, marshalled).Err()
	if err != nil {
		return
	}

	_, err = s.ChannelMessageSendEmbed(channel.ID, embed)
	if err != nil {
		fmt.Println("Error sending Embed DM:", err)
		return
	}
}

var botCategory string = "1359830076455125185"

func ReactionHandler(s *discordgo.Session, i *discordgo.MessageReactionAdd) {
	chanIn, chanOut := sessions.SessionExists(i.Member.User.ID)
	if chanIn != nil {
		msg, err := s.ChannelMessage(i.ChannelID, i.MessageID)
		if err != nil {
			fmt.Println("Error fetching message:", err)
			return
		}
		chanIn <- 2
		sessionCopy := <-chanOut
		if sessionCopy.CurrentApp != nil {
			if msg.Author.ID == s.State.User.ID && i.MessageID == sessionCopy.CurrentApp.EmbedID && i.UserID != s.State.User.ID {
				var state bool
				if i.Emoji.Name == "✅" {
					state = true
					sessionCopy.CurrentApp.App.Verdict = &state
					handleApplicationDecision(s, true, *sessionCopy.CurrentApp.App)
					chanIn <- 1
				} else if i.Emoji.Name == "❌" {
					state = false
					sessionCopy.CurrentApp.App.Verdict = &state
					handleApplicationDecision(s, false, *sessionCopy.CurrentApp.App)
					chanIn <- 1
				}
			}
		}
	}
}

func CommandsHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "submit":

			var message string
			if len(i.ApplicationCommandData().Options) > 0 {
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
			Session, _ := sessions.SessionOpen(s, i)
			sessions.EnterLoop(Session)
		case "listapps":
			for _, appgroup := range Applications {
				fmt.Println(appgroup)
			}
		case "clearapps":
			Applications = make(AllApps)
			err := rds.Del("applications").Err()
			if err != nil {
				return
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

	intents := discordgo.IntentsGuildInvites | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessageReactions
	dg.Identify.Intents = intents

	dg.AddHandler(ready)
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

		for uid, appgroup := range Applications {
			err := rds.HSet("applications", uid, appgroup).Err()
			fmt.Println(uid, appgroup)
			if err != nil {
				fmt.Printf("Failed to set application for user %s: %v", uid, err)
			}
		}

		dg.Close()
		s.Close()
	}(dg)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
