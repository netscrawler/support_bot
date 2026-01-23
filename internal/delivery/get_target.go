package delivery

import (
	"errors"

	models "support_bot/internal/models/report"
)

func GetTarget(recipients ...models.Recipient) ([]models.Targeted, error) {
	var (
		targets []models.Targeted
		err     error
	)

	for _, r := range recipients {
		switch r.Type {
		case models.EmailRecipient:
			if r.Email != nil {
				email, tErr := models.NewTargetEmail(*r.Email)
				if tErr != nil {
					err = errors.Join(err, tErr)
				}

				targets = append(targets, email)
			}

		case models.SambaRecipient:
			if r.RemotePath != nil {
				targets = append(targets, models.TargetFileServer{Dest: *r.RemotePath})
			}
		case models.TelegramRecipient:
			targets = append(targets, models.NewTargetTelegramChat(r.Chat.ChatID, r.ThreadID))
		default:
		}
	}

	return targets, nil
}
