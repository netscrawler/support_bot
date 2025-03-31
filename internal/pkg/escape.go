package pkg

import "strings"

func EscapeMarkdownV2(text string) string {
	escapeChars := "_[]()~`>#+-=|{}.!"
	for _, char := range escapeChars {
		text = strings.ReplaceAll(text, string(char), "\\"+string(char))
	}
	return text
}
