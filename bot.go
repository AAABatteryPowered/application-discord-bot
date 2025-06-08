package main

import (
	"bot/giveaways"
	"bot/levels"
	"bot/redis"
	"bot/utils"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var token string
var guildid string

var CooldownCache map[string]int = make(map[string]int)

func onStartup(s *discordgo.Session, r *discordgo.Ready) {
	RegisterCommands(s)
}

func RegisterCommands(s *discordgo.Session) {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "applybutton",
			Description: "Sends the apply button message",
		},
		{
			Name:        "level",
			Description: "Tells you what level you are and your xp progress to the next.",
		},
		{
			Name:        "giveaway",
			Description: "All of the subcommands for the giveaway feature.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "create",
					Description: "Creates a new giveaway",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "prize",
							Description: "What's the prize?",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "duration",
							Description: "Duration in minutes",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "winners",
							Description: "Number of winners",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "end",
					Description: "End a giveaway early",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "id",
							Description: "Giveaway ID",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List active giveaways",
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

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if m.Content == "!reactionroles" {
		// Delete user's message (optional)
		err := s.ChannelMessageDelete(m.ChannelID, m.ID)
		if err != nil {
			fmt.Printf("Failed to delete command message: %v", err)
		}

		// Send embed
		msg, err := s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       "Reaction Roles",
			Description: "🚀 <@&1373246088978628733>\n🎁 <@&1373246065658036284>\n📊 <@&1373246111107514450>\n🌊 <@&1373245945453482075>",
			Color:       0x00ff9d,
		})
		if err != nil {
			fmt.Printf("Failed to send embed: %v", err)
			return
		}

		// Add reactions
		emojis := []string{"🚀", "🎁", "📊", "🌊"}
		for _, emoji := range emojis {
			err := s.MessageReactionAdd(m.ChannelID, msg.ID, emoji)
			if err != nil {
				fmt.Printf("Failed to add emoji %s: %v", emoji, err)
			}
		}
	}
}

func CommandsHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "applybutton":
			if i.Member.User.ID != "1113062986718908526" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You do not have permission to do this!",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			button := discordgo.Button{
				Label:    "Create an application",
				Style:    discordgo.PrimaryButton,
				CustomID: "application_form_open_button",
				Emoji: &discordgo.ComponentEmoji{
					Name: "📝",
				},
			}

			actionRow := discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{button},
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Components: []discordgo.MessageComponent{actionRow},
				},
			})
		}
	}
	if i.Type == discordgo.InteractionMessageComponent {
		switch i.MessageComponentData().CustomID {
		case "application_form_open_button":
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

var reactionRoleMap = map[string]string{
	"🚀": "1373246088978628733",
	"🎁": "1373246065658036284",
	"🌊": "1373245945453482075",
	"📊": "1373246111107514450",
}

func handlReactionAdded(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.Member.User.ID != s.State.User.ID {

		if r.MessageID == "1373250549968797796" {
			roleID, ok := reactionRoleMap[r.Emoji.Name]
			if !ok {
				return
			}

			err := s.GuildMemberRoleAdd(r.GuildID, r.UserID, roleID)
			if err != nil {
				fmt.Printf("Failed to add role: %v", err)
			}
			return
		}

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

func handlReactionRemoved(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.UserID == s.State.User.ID {
		return
	}

	if r.MessageID == "1373250549968797796" {
		roleID, ok := reactionRoleMap[r.Emoji.Name]
		if !ok {
			return
		}

		err := s.GuildMemberRoleRemove(r.GuildID, r.UserID, roleID)
		if err != nil {
			fmt.Printf("Failed to remove role: %v", err)
		}
	}
}

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Errorf("Error loading .env file: %s", err)
	}

	token = os.Getenv("TOKEN")
	guildid = os.Getenv("GUILDID")

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session,", err)
		return
	}

	redis.InitRedis()

	intents := discordgo.IntentsGuildInvites | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessageReactions
	dg.Identify.Intents = intents

	dg.AddHandler(onStartup)
	dg.AddHandler(CommandsHandler)
	dg.AddHandler(handlInteractionCreate)
	dg.AddHandler(handlReactionAdded)
	dg.AddHandler(handlReactionRemoved)
	dg.AddHandler(messageCreate)

	levels.Start(dg)
	giveaways.Start(dg)

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
