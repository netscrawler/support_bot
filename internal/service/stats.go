package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"support_bot/internal/models"
	"support_bot/internal/pkg/xlsx"
	"text/template"
	"time"

	pngpkg "support_bot/internal/pkg/png"

	"github.com/robfig/cron/v3"
	"gopkg.in/telebot.v4"
)

type StatsQueryGetter interface {
	GetAllActive(ctx context.Context) ([]models.Notify, error)
}

type MetabaseDataGetter interface {
	GetDataMatrix(ctx context.Context, cardUUID string) ([][]string, error)
	GetDataMap(ctx context.Context, cardUUID string) (map[string]any, error)
}

type Stats struct {
	query    StatsQueryGetter
	message  MessageSender
	metabase MetabaseDataGetter
	cron     *cron.Cron
}

func New(q StatsQueryGetter, mess MessageSender, mb MetabaseDataGetter) *Stats {
	return &Stats{
		query:    q,
		message:  mess,
		metabase: mb,
		cron:     cron.New(),
	}
}

type CronJobs struct {
	Total    int
	Success  int
	Unsucess map[string]error
}

// Start запускает крон-задачи для всех активных уведомлений.
func (s *Stats) Start(ctx context.Context) (CronJobs, error) {
	c := CronJobs{
		Total:    0,
		Success:  0,
		Unsucess: make(map[string]error),
	}
	notifies, err := s.query.GetAllActive(ctx)
	if err != nil {
		return c, fmt.Errorf("failed to get notifies: %w", err)
	}

	logger := slog.Default()

	// Останавливаем предыдущие задачи
	s.cron.Stop()

	// Создаем новый планировщик
	s.cron = cron.New()

	// Группируем уведомления по GroupID
	groupedNotifies := make(map[string][]models.Notify)
	groupCron := make(map[string]string)

	for _, notify := range notifies {
		groupID := notify.GroupID
		if groupID == "" {
			groupID = "default" // Если GroupID пустой, используем default
		}

		groupedNotifies[groupID] = append(groupedNotifies[groupID], notify)
		groupCron[groupID] = string(notify.Cron)
	}

	// Создаем крон-задачи для каждой группы
	for groupID, groupNotifies := range groupedNotifies {
		c.Total++
		_, err := s.cron.AddFunc(groupCron[groupID], func() {
			s.sendGroupNotifications(ctx, groupNotifies)
		})
		if err != nil {
			c.Unsucess[groupID] = err
			logger.ErrorContext(ctx, "failed to add cron job for group",
				slog.String("groupID", groupID),
				slog.Any("error", err))

			continue
		}
		c.Success++

		logger.InfoContext(ctx, "added cron job for group",
			slog.String("groupID", groupID),
			slog.Int("notifyCount", len(groupNotifies)))
	}

	// Запускаем планировщик
	s.cron.Start()

	logger.InfoContext(ctx, "started cron scheduler",
		slog.Int("activeJobs", len(s.cron.Entries())))

	return c, nil
}

// Stop останавливает все крон-задачи.
func (s *Stats) Stop() {
	if s.cron != nil {
		slog.Info("stopping crong jobs")
		s.cron.Stop()
	}
}

func (s *Stats) fillTemplateWithData(templateStr string, data map[string]any) string {
	tmpl, err := template.New("").Parse(templateStr)
	if err != nil {
		slog.Error("unable parse template", slog.Any("error", err))
		return templateStr
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		slog.Error("unable fill template", slog.Any("error", err))
		return templateStr
	}
	return buf.String()
}

type chat struct {
	ChatID   int64
	ThreadID int64
}

// sendGroupNotifications отправляет группу уведомлений.
func (s *Stats) sendGroupNotifications(ctx context.Context, notifies []models.Notify) {
	logger := slog.Default()

	// Группируем уведомления по ChatID
	chatGroups := make(map[chat][]models.Notify)
	for _, notify := range notifies {
		c := chat{ChatID: notify.ChatID, ThreadID: notify.ThreadID}
		chatGroups[c] = append(chatGroups[c], notify)
	}

	// Отправляем каждому чату
	for chatID, chatNotifies := range chatGroups {
		err := s.sendChatNotifications(ctx, chatID, chatNotifies)
		if err != nil {
			logger.ErrorContext(ctx, "failed to send chat notifications",
				slog.Int64("chatID", chatID.ChatID),
				slog.Int64("threadID", chatID.ThreadID),
				slog.Any("error", err))
		}
	}
}

// sendChatNotifications отправляет все уведомления в один чат.
func (s *Stats) sendChatNotifications(
	ctx context.Context,
	target chat,
	notifies []models.Notify,
) error {
	logger := slog.Default()

	chat := &telebot.Chat{ID: target.ChatID}

	var pngImages []*bytes.Buffer

	xlsxData := make(map[string][][]string)
	groupName := ""

	var csvData []*bytes.Buffer

	var textMessages []string

	for _, notify := range notifies {

		groupName, _ = notify.GetGroupTitle()
		data, err := s.metabase.GetDataMatrix(ctx, notify.CardUUID)
		if err != nil {
			logger.ErrorContext(ctx, "failed to get metabase data",
				slog.String("name", notify.Name),
				slog.String("query", notify.CardUUID),
				slog.Any("error", err))

			continue
		}

		for _, format := range notify.NotifyFormat {
			switch format {
			case models.NotifyFormatPng:
				// Создаем PNG изображение
				pngName := notify.Title + time.Now().
					Add(-time.Hour*24).
					Format("02-01-2006") +
					".png"

				img, err := pngpkg.CreateImageFromMatrix(data, pngName, notify.Title)
				if err != nil {
					logger.ErrorContext(ctx, "error creating png", slog.Any("error", err))
				}

				pngImages = append(pngImages, img)

			case models.NotifyFormatXlsx:
				// Создаем XLSX файл
				xlsxData[notify.Title] = data

			case models.NotifyFormatText:
				// Форматируем текстовое сообщение
				jsonData, err := s.metabase.GetDataMap(ctx, notify.CardUUID)
				if err != nil {
					logger.ErrorContext(ctx, "failed to format text message",
						slog.String("name", notify.Name),
						slog.Any("error", err))

					continue
				}

				textMsg := s.fillTemplateWithData(*notify.TemplateText, jsonData)

				textMessages = append(textMessages, textMsg)

			case models.NotifyFormatCsv:
				csv := s.formatDataAsCSV(data)

				csvData = append(csvData, csv)

			}
		}
	}

	return s.sendData(
		ctx,
		xlsxData,
		csvData,
		textMessages,
		pngImages,
		groupName,
		chat,
		target.ThreadID,
	)
}

func (s *Stats) sendData(
	ctx context.Context,
	xlsxData map[string][][]string,
	csvData []*bytes.Buffer,
	textMessages []string,
	pngImages []*bytes.Buffer,
	title string,
	chat *telebot.Chat,
	threadID int64,
) error {
	logger := slog.Default()
	if len(pngImages) > 0 {
		err := s.message.SendMedia(chat, pngImages)
		if err != nil {
			logger.ErrorContext(ctx, "failed to send png caption", slog.Any("error", err))
		}
	}

	if len(xlsxData) > 0 {
		xlsxBook, err := xlsx.CreateXlsxBook(xlsxData)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create xlsx book", slog.Any("error", err))

			return err
		}

		if err = s.message.SendDocument(chat, xlsxBook, title+".xlsx"); err != nil {
			logger.ErrorContext(ctx, "failed to send xlsx file", slog.Any("error", err))

			return err
		}

	}

	for _, textMsg := range textMessages {
		err := s.message.Send(chat, textMsg, &telebot.SendOptions{
			ParseMode: telebot.ModeMarkdown,
			ThreadID:  int(threadID),
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to send text message", slog.Any("error", err))
		}
	}

	if len(csvData) > 0 {
		err := s.message.SendDocument(
			chat,
			csvData[0],
			title+".csv",
			&telebot.SendOptions{ThreadID: int(threadID)},
		)
		if err != nil {
			logger.ErrorContext(ctx, "failed to send csv file", slog.Any("error", err))
		}
	}

	return nil
}

// formatDataAsCSV форматирует данные как CSV.
func (s *Stats) formatDataAsCSV(data [][]string) *bytes.Buffer {
	if len(data) == 0 {
		return nil
	}

	var buf bytes.Buffer

	r := csv.NewWriter(&buf)
	r.WriteAll(data)

	return &buf
}
