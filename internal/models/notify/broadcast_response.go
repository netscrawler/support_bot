package models

import (
	"fmt"
	"strings"

	"support_bot/internal/pkg"
)

type BroadcastResp struct {
	ChatsCount       int
	SuccessCount     int
	ErrorCount       int
	ErrorChatsTitles []string
}

func NewBroadcastResp() *BroadcastResp {
	return &BroadcastResp{
		ChatsCount:       0,
		SuccessCount:     0,
		ErrorCount:       0,
		ErrorChatsTitles: []string{},
	}
}

func (br *BroadcastResp) AddSuccess() {
	br.ChatsCount++
	br.SuccessCount++
}

func (br *BroadcastResp) AddError(chatName string, err error) {
	br.ChatsCount++
	br.ErrorCount++
	br.ErrorChatsTitles = append(br.ErrorChatsTitles, "- "+chatName, " : ", err.Error())
}

func (br *BroadcastResp) String() string {
	formattedMsg := fmt.Sprintf(
		"✅ *Уведомления отправлены*\n\n"+
			"Всего чатов: *%d*\n"+
			"Успешно: *%d*\n\n",
		br.ChatsCount, br.SuccessCount,
	)

	if br.ErrorCount != 0 {
		formattedMsg += fmt.Sprintf(
			"Не отправлено: *%d*\n"+
				"В чаты: %s/n"+
				"Note: Пожалуйста, проверьте, есть ли какие-либо особые проблемы в неудачных чатах.",
			br.ErrorCount,
			strings.Join(br.ErrorChatsTitles, "\n"),
		)
	}

	return pkg.EscapeMarkdownV2(formattedMsg)
}
