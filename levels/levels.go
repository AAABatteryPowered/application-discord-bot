package levels

import (
	botredis "bot/redis"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
)

var MessageCooldowns map[string]int64 = make(map[string]int64)
var XPTable map[string]int = GenerateXPTable()

func GenerateXPTable() map[string]int {
	a, b, c := 10, 50, 100
	xpTable := make(map[string]int)
	totalXP := 0

	for level := 1; level <= 50; level++ {
		xpToNext := a*(level-1)*(level-1) + b*(level-1) + c
		totalXP += xpToNext
		xpTable[fmt.Sprintf("%d", level)] = totalXP
	}

	return xpTable
}

func GetLevelFromXP(userXP int) int {
	for level := 1; level <= 50; level++ {
		levelKey := fmt.Sprintf("%d", level)
		if userXP < XPTable[levelKey] {
			return level - 1
		}
	}
	return 50
}

var levelToRole map[int]string = map[int]string{
	1:  "1373281910511501463",
	5:  "1373282115218575451",
	10: "1373282162593501276",
	15: "1373282207095066694",
	20: "1373282279731761232",
	25: "1373282351169404978",
	30: "1373282418542379059",
	50: "1380853553467359252",
}

func doesLevelHaveRole(num int) bool {
	return num == 1 || (num%5 == 0 && num <= 30) || num == 50
}

func awardLevel(s *discordgo.Session, memberid string, level int) {
	lvlUpMessage := fmt.Sprintf("<@%s> has reached level %d. Congrats!", memberid, level)
	_, err := s.ChannelMessageSend("1373260159899537479", lvlUpMessage)
	if err != nil {
		fmt.Printf("Error sending message: %v", err)
	}

	idint, err := strconv.Atoi(memberid)
	if err != nil {
		fmt.Printf("Invalid number: %v\n", err)
	}

	if doesLevelHaveRole(level) {
		s.GuildMemberRoleAdd(os.Getenv("GUILDID"), fmt.Sprintf("%d", idint), levelToRole[level])
		if err != nil {
			fmt.Printf("Error assigning role: %v", err)
		}
	}
}

var roleXpMultipliers map[string]float32 = map[string]float32{
	"1373313471919423598": 1.5,
}

func calculateXpGain(message *discordgo.Message, sender *discordgo.Member) int {
	var totalxp float32 = 0

	for _, attachment := range message.Attachments {
		contentType := strings.ToLower(attachment.ContentType)
		filename := strings.ToLower(attachment.Filename)

		if strings.HasPrefix(contentType, "image/") {
			if strings.Contains(contentType, "gif") || strings.HasSuffix(filename, ".gif") {
				totalxp += 2
			} else {
				totalxp += 3
			}
		} else {
			totalxp += 1
		}
	}

	if len(message.Content) > 0 && message.Content != "" {
		totalxp += 4
	}

	for _, role := range sender.Roles {
		multiplier, exists := roleXpMultipliers[role]
		if exists {
			totalxp *= multiplier
		}
	}

	return int(totalxp)
}

func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	cooldown, exists := MessageCooldowns[m.Author.ID]
	var cooldowndifference bool
	if exists {
		cooldowndifference = time.Now().Unix()-cooldown > 3
	}

	if cooldowndifference || !exists {
		MessageCooldowns[m.Author.ID] = time.Now().Unix()

		currentexpstr, err := botredis.RedisC.HGet("levelsxp", m.Author.ID).Result()
		if err != nil {
			if err == redis.Nil {
				_, err := botredis.RedisC.HSet("levelsxp", m.Author.ID, 0).Result()
				if err != nil {
					fmt.Printf("Error setting XP: %v\n", err)
					return
				}
			} else {
				fmt.Println(fmt.Sprintf("redis nil?%s", err))
			}
			return
		}

		currentexp, err := strconv.Atoi(currentexpstr)
		if err != nil {
			fmt.Printf("Field has non-integer value")
			return
		}

		boost := calculateXpGain(m.Message, m.Member)
		err = botredis.RedisC.HSet("levelsxp", m.Author.ID, currentexp+boost).Err()
		if err != nil {
			fmt.Println(err)
			return
		}

		if GetLevelFromXP(currentexp+boost) > GetLevelFromXP(currentexp) {
			awardLevel(s, m.Author.ID, GetLevelFromXP(currentexp+boost))
		}
	}
}

func onLevelCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "level":
			currentXPStr, err := botredis.RedisC.HGet("levelsxp", i.Member.User.ID).Result()
			if err != nil {
				if err == redis.Nil {
					_, err := botredis.RedisC.HSet("levelsxp", i.Member.User.ID, 0).Result()
					if err != nil {
						s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Content: "You have no xp! Your level is 1.",
								Flags:   discordgo.MessageFlagsEphemeral,
							},
						})
						return
					}
					currentXPStr = "0"
				} else {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "There was an error retrieving your level. Please try again, and if the error persists create a support ticket.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}
			}

			currentXP, err := strconv.Atoi(currentXPStr)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Oops, an error occured!",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			currentLevel := GetLevelFromXP(currentXP)
			xpProgress := currentXP - XPTable[fmt.Sprintf("%d", currentLevel)]
			levelGap := XPTable[fmt.Sprintf("%d", currentLevel+1)] - XPTable[fmt.Sprintf("%d", currentLevel)]

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("You are at Level **%d** and your progress to Level %d is **%d/%d** Experience.", currentLevel, currentLevel+1, xpProgress, levelGap),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
	}
}

func Start(dg *discordgo.Session) {
	dg.AddHandler(onMessage)
	dg.AddHandler(onLevelCommands)
}
