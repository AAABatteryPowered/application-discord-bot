package giveaways

import (
	botredis "bot/redis"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

type Giveaway struct {
	Prize        string
	Duration     int
	CreationTime int64
	Winners      int
	Creator      string
	Participants []string
	MessageID    string
}

var giveawayChannel int = 1373209434049740912
var giveaways map[string]Giveaway

func createGiveaway(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var prize string
	var duration int
	var winners int = 1

	for _, option := range options {
		switch option.Name {
		case "prize":
			prize = option.StringValue()
		case "duration":
			duration = int(option.IntValue())
		case "winners":
			if int(option.IntValue()) > 0 {
				winners = int(option.IntValue())
			}
		}
	}

	logTime := time.Now().Unix()
	endTime := logTime + int64(duration*60)

	embed := &discordgo.MessageEmbed{
		Title:       prize,
		Description: fmt.Sprintf("Ends in: %s (%s)\nHosted by: %s\nWinners: %d", fmt.Sprintf("<t:%d:R>", endTime), fmt.Sprintf("<t:%d:f>", endTime), fmt.Sprintf("<@%s>", i.Member.User.ID), winners),
		Color:       0x5496ff,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Hosted by %s", i.Member.User.Username),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	var giveawayID string

	for attempts := 0; attempts < 10; attempts++ {
		giveawayID = uuid.New().String()

		exists, err := botredis.RedisC.HExists("giveaways", giveawayID).Result()
		if err != nil {
			fmt.Errorf("error checking giveaway ID: %v", err)
			continue
		}

		if exists {
			fmt.Errorf("UUID collision occurred")
			continue
		}
		break
	}

	if giveawayID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Wow, you got unlucky! Failed to generate a UUID 10 times.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	enterButton := discordgo.Button{
		Label:    "🎉 Enter Giveaway",
		Style:    discordgo.PrimaryButton,
		CustomID: fmt.Sprintf("giveaway_enter_%s", giveawayID),
	}

	actionRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{enterButton},
	}

	embedmessage, err := s.ChannelMessageSendComplex(fmt.Sprintf("%d", giveawayChannel), &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{actionRow},
	})

	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Error sending giveaway: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	giveawayObject := Giveaway{
		Prize:        prize,
		Duration:     duration,
		CreationTime: logTime,
		Winners:      winners,
		Creator:      i.Member.User.ID,
		Participants: 0,
		MessageID:    embedmessage.ID,
	}

	giveaways[giveawayID] = giveawayObject

	giveawayJSON, err := json.Marshal(giveawayObject)
	if err != nil {
		fmt.Printf("error marshaling giveaway: %v", err)
		return
	}

	err = botredis.RedisC.HSet("giveaways", giveawayID, giveawayJSON).Err()
	if err != nil {
		fmt.Printf("error setting giveaway in Redis: %v", err)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Giveaway created successfully in <#%d>!", giveawayChannel),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func enterGiveaway(s *discordgo.Session, i *discordgo.InteractionCreate, giveawayID string) {
	giveaway, exists := giveaways[giveawayID]
	if !exists {
		return
	}

	giveaway.Participants = append(giveaway.Participants, i.Member.User.ID)

	updatedembed := &discordgo.MessageEmbed{
		Title:       giveaway.Prize,
		Description: fmt.Sprintf("Ends in: %s (%s)\nHosted by: %s\nWinners: %d", fmt.Sprintf("<t:%d:R>", giveaway.CreationTime+int64(giveaway.Duration*60)), fmt.Sprintf("<t:%d:f>", giveaway.CreationTime+int64(giveaway.Duration*60)), fmt.Sprintf("<@%s>", giveaway.Creator), giveaway.Winners),
		Color:       0x5496ff,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Hosted by %s", i.Member.User.Username),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	edit := &discordgo.MessageEdit{
		Embeds: []*discordgo.MessageEmbed{updatedembed},
	}

	_, err := s.ChannelMessageEditComplex(edit)
	if err != nil {
		return
	}
}

func onGiveawayCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "giveaway":

			data := i.ApplicationCommandData()
			if len(data.Options) == 0 {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "⚠️ Uh oh! It looks like you need **subcommands** for this command.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return // No subcommand provided
			}

			subcommand := data.Options[0]

			// Handle different subcommands
			switch subcommand.Name {
			case "create":
				createGiveaway(s, i, subcommand.Options)
			}
		}
	}
	if i.Type == discordgo.InteractionMessageComponent {
		customID := i.MessageComponentData().CustomID

		if strings.HasPrefix(customID, "giveaway_enter_") {
			giveawayID := strings.TrimPrefix(customID, "giveaway_enter_")
			enterGiveaway(s, i, giveawayID)
		}
	}
}

func Start(dg *discordgo.Session) {
	dg.AddHandler(onGiveawayCommands)
}
