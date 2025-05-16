package main

import (
	"bot/utils"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
)

// Bot Token (You need to replace this with your bot's token)
const token = "MTMyNzU4Mjg0MTkwNjc5NDU1Ng.GsPPqZ.YSbhP_b0wcJ-PHRmOXTl9PNwySJWrOAC2kFmxI"
const guildid = "1355623019971608706"

var rds *redis.Client

var CooldownCache map[string]int = make(map[string]int)

func onStartup(s *discordgo.Session, r *discordgo.Ready) {
	RegisterCommands(s)
	//applications.OnStart(rds)
}

func RegisterCommands(s *discordgo.Session) {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "applybutton",
			Description: "Sends the apply button message",
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
		case "applybutton":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &discordgo.InteractionResponseData{
					CustomID: "submit_application" + i.Interaction.Member.User.ID,
					Title:    "Submit your application",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    "app_link",
									Label:       "Application Link",
									Style:       discordgo.TextInputShort,
									Placeholder: "e.g https://www.youtube.com/",
									Required:    true,
									MaxLength:   300,
									MinLength:   4,
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    "join_reason",
									Label:       "Why do you want to join?",
									Placeholder: "e.g I need somewhere to make content.",
									Style:       discordgo.TextInputParagraph,
									Required:    true,
									MinLength:   30,
									MaxLength:   500,
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    "unique_offer",
									Label:       "What can you offer that NOBODY else can?",
									Placeholder: "e.g I'm really skilled at singing, so I could do a kareoke event on the server.",
									Style:       discordgo.TextInputParagraph,
									Required:    true,
									MinLength:   30,
									MaxLength:   500,
								},
							},
						},
					},
				},
			})
		default:
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Sorry, I couldn't recognize this command!",
					//Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
	}
}

func handlInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionModalSubmit:
		if i.ModalSubmitData().CustomID == "submit_application"+i.Interaction.Member.User.ID {
			data := i.ModalSubmitData()
			var joinreason string = data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
			var unique_offer string = data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
			var video_link string = data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
			_, err := url.ParseRequestURI(video_link)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You need to enter a youtube url!",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			threadData := &discordgo.ThreadStart{
				Name:                fmt.Sprintf("%v's Application", i.Member.User.GlobalName), // Title of the forum post
				AutoArchiveDuration: 60,                                                        // Auto-archive duration in minutes
				Type:                discordgo.ChannelTypeGuildPublicThread,
				AppliedTags:         []string{"1368576663608360981"}, // Optional: Tag IDs
			}

			now := time.Now()
			day := now.Day()
			suffix := utils.OrdinalSuffix(day)
			month := now.Month().String()
			year := now.Year()
			hour := now.Format("15:04") // 24-hour format

			formatted := fmt.Sprintf("<@%v>\nThis application was submitted on the %d%s of %s %d at %s\n%v", i.Interaction.Member.User.ID, day, suffix, month, year, hour, video_link)

			messageData := &discordgo.MessageSend{
				Content: formatted,
				Embeds: []*discordgo.MessageEmbed{
					{
						URL:   video_link,
						Color: 0x00ff00,
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:   "Reason for Joining:",
								Value:  joinreason,
								Inline: false,
							},
							{
								Name:   "What only they can offer:",
								Value:  unique_offer,
								Inline: false,
							},
						},
					},
				},
			}

			thread, err := s.ForumThreadStartComplex("1368576531756220447", threadData, messageData)
			if err != nil {
				fmt.Println("Error creating forum post:", err)
				return
			}
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Your Application Thread has been created in <#%v>", thread.ID),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				panic(err)
			}

			//Reactions
			err = s.MessageReactionAdd(thread.ID, thread.LastMessageID, "✅")
			if err != nil {
				fmt.Println("Failed to react:", err)
			}
			err = s.MessageReactionAdd(thread.ID, thread.LastMessageID, "❌")
			if err != nil {
				fmt.Println("Failed to react:", err)
			}
		}
	}
}

func handlReactionAdded(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.Member.User.ID != s.State.User.ID {
		hasRole := false
		for _, roleID := range r.Member.Roles {
			fmt.Println(roleID)
			if roleID == "1358026605330563185" {
				hasRole = true
				break
			}
		}
		if !hasRole {
			err := s.MessageReactionRemove(r.ChannelID, r.MessageID, "✅", r.UserID)
			if err != nil {
				fmt.Println("Failed to remove reaction:", err)
			}
			return
		}

		botmessage, err := s.ChannelMessage(r.ChannelID, r.MessageID)
		if err != nil {
			fmt.Println("Error fetching message:", err)
			return
		}

		ping := strings.SplitN(botmessage.Content, "\n", 2)[0]
		cleaned := strings.Map(func(r rune) rune {
			if r == '<' || r == '>' || r == '@' {
				return -1
			}
			return r
		}, ping)

		dmchannel, err := s.UserChannelCreate(cleaned)
		if err != nil {
			fmt.Println("Error creating DM channel:", err)
			return
		}

		var embed *discordgo.MessageEmbed

		tr := true
		if r.Emoji.Name == "❌" {
			newtags := []string{"1368861171939147796"}
			_, err := s.ChannelEditComplex(r.ChannelID, &discordgo.ChannelEdit{
				Archived:    &tr,
				Locked:      &tr,
				AppliedTags: &newtags,
			})
			if err != nil {
				fmt.Println("Error locking and archiving thread:", err)
			}
			embed = &discordgo.MessageEmbed{
				Title:       "Your application has been denied ❌",
				Description: "Unfortunately, the application reviewing team has decided to **deny you** from the Midas SMP. Your best bet to get accepted is to reread the application rules/standards, learn some new skills and submit another new and improved application.",
				Color:       0xff4070,
				Footer: &discordgo.MessageEmbedFooter{
					Text: "- Midas SMP Application Reviewers",
				},
			}
		} else if r.Emoji.Name == "✅" {
			newtags := []string{"1368861046550429727"}
			_, err := s.ChannelEditComplex(r.ChannelID, &discordgo.ChannelEdit{
				Archived:    &tr,
				Locked:      &tr,
				AppliedTags: &newtags,
			})
			if err != nil {
				fmt.Println("Error locking and archiving thread:", err)
			}
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
		}
		_, err = s.ChannelMessageSendEmbed(dmchannel.ID, embed)
		if err != nil {
			fmt.Println("Error sending Embed DM:", err)
			return
		}
	}
}

func InitRedis() {
	rds = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	_, err := rds.Ping().Result()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("#[Redis]: Started Successfully!")
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

	dg.AddHandler(onStartup)
	dg.AddHandler(CommandsHandler)
	dg.AddHandler(handlInteractionCreate)
	dg.AddHandler(handlReactionAdded)

	err = dg.Open()
	if err != nil {
		fmt.Println("#[Main]: Error opening discordgo connection: ", err)
		return
	}
	fmt.Println("#[Main]: Bot is running successfully!")

	// Shutdown

	defer func(s *discordgo.Session) {
		fmt.Println("#[Main]: Starting shutdown logic.")

		dg.Close()
		s.Close()
	}(dg)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
