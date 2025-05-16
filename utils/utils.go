package utils

import (
	"fmt"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

func ExtractYouTubeID(url string) string {
	re := regexp.MustCompile(`(?:https?:\/\/(?:www\.)?youtube\.com\/(?:[^\/\n\s]+\/\S+\/|\S+\?v=|(?:v|e(?:mbed)?)\/|(?:.*[?&]v=))([^""&?\/\s]{11}))`)
	match := re.FindStringSubmatch(url)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func GetChannelInCategoryByName(s *discordgo.Session, guildID string, categoryID string, name string) (*discordgo.Channel, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch channels: %w", err)
	}

	for _, ch := range channels {
		if ch.ParentID == categoryID && ch.Name == name {
			return ch, nil
		}
	}

	return nil, nil
}

func OrdinalSuffix(day int) string {
	if day >= 11 && day <= 13 {
		return "th"
	}
	switch day % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}
