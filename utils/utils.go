package utils

import "regexp"

func ExtractYouTubeID(url string) string {
	re := regexp.MustCompile(`(?:https?:\/\/(?:www\.)?youtube\.com\/(?:[^\/\n\s]+\/\S+\/|\S+\?v=|(?:v|e(?:mbed)?)\/|(?:.*[?&]v=))([^""&?\/\s]{11}))`)
	match := re.FindStringSubmatch(url)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}
