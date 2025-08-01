package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"support_bot/internal/models"
	"support_bot/internal/pkg/xlsx"
	"time"

	pngpkg "support_bot/internal/pkg/png"

	"github.com/robfig/cron/v3"
	"gopkg.in/telebot.v4"
)

type StatsQueryGetter interface {
	GetAll(ctx context.Context) ([]models.Notify, error)
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

// Start запускает крон-задачи для всех активных уведомлений.
func (s *Stats) Start(ctx context.Context) error {
	notifies, err := s.query.GetAllActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to get notifies: %w", err)
	}

	logger := slog.Default()

	// Останавливаем предыдущие задачи
	s.cron.Stop()

	// Создаем новый планировщик
	s.cron = cron.New()

	// Группируем уведомления по GroupID
	groupedNotifies := make(map[string][]models.Notify)

	for _, notify := range notifies {
		groupID := notify.GroupID
		if groupID == "" {
			groupID = "default" // Если GroupID пустой, используем default
		}

		groupedNotifies[groupID] = append(groupedNotifies[groupID], notify)
	}

	// Создаем крон-задачи для каждой группы
	for groupID, groupNotifies := range groupedNotifies {
		_, err := s.cron.AddFunc("* * * * *", func() {
			s.sendGroupNotifications(ctx, groupNotifies)
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to add cron job for group",
				slog.String("groupID", groupID),
				slog.Any("error", err))

			continue
		}

		logger.InfoContext(ctx, "added cron job for group",
			slog.String("groupID", groupID),
			slog.Int("notifyCount", len(groupNotifies)))
	}

	// Запускаем планировщик
	s.cron.Start()

	logger.InfoContext(ctx, "started cron scheduler",
		slog.Int("activeJobs", len(s.cron.Entries())))

	return nil
}

// Stop останавливает все крон-задачи.
func (s *Stats) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}

// getMetabaseData получает данные из Metabase.
func (s *Stats) getMetabaseData(ctx context.Context, notify models.Notify) ([][]string, error) {
	if s.metabase == nil {
		return nil, errors.New("metabase data getter not implemented")
	}

	return s.metabase.GetDataMatrix(ctx, notify.CardUUID)
}

// fillTemplateWithData заполняет шаблон данными из Metabase.
func (s *Stats) fillTemplateWithData(template string, data map[string]any) string {
	// Простая реализация замены плейсхолдеров
	// Можно использовать более сложные библиотеки для шаблонизации
	result := template

	// Заменяем плейсхолдеры вида {{key}} на значения из data
	for key, value := range data {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		valueStr := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, valueStr)
	}

	return result
}

// formatDataAsText форматирует данные как текстовую таблицу.
func (s *Stats) formatDataAsText(data [][]string) string {
	if len(data) == 0 {
		return "Нет данных для отображения"
	}

	var result strings.Builder

	result.WriteString("📋 *Данные:*\n\n")

	// Определяем максимальную ширину для каждой колонки
	columnWidths := make([]int, len(data[0]))

	for _, row := range data {
		for i, cell := range row {
			if len(cell) > columnWidths[i] {
				columnWidths[i] = len(cell)
			}
		}
	}

	// Выводим заголовки (первая строка)
	if len(data) > 0 {
		result.WriteString("| ")

		for i, cell := range data[0] {
			result.WriteString(fmt.Sprintf("%-*s | ", columnWidths[i], cell))
		}

		result.WriteString("\n")

		// Разделитель
		result.WriteString("|")

		for _, width := range columnWidths {
			result.WriteString(strings.Repeat("-", width+2))
			result.WriteString("|")
		}

		result.WriteString("\n")

		// Данные
		for _, row := range data[1:] {
			result.WriteString("| ")

			for i, cell := range row {
				result.WriteString(fmt.Sprintf("%-*s | ", columnWidths[i], cell))
			}

			result.WriteString("\n")
		}
	}

	return result.String()
}

// sendGroupNotifications отправляет группу уведомлений.
func (s *Stats) sendGroupNotifications(ctx context.Context, notifies []models.Notify) {
	logger := slog.Default()

	// Группируем уведомления по ChatID
	chatGroups := make(map[int64][]models.Notify)
	for _, notify := range notifies {
		chatGroups[notify.ChatID] = append(chatGroups[notify.ChatID], notify)
	}

	// Отправляем каждому чату
	for chatID, chatNotifies := range chatGroups {
		err := s.sendChatNotifications(ctx, chatID, chatNotifies)
		if err != nil {
			logger.ErrorContext(ctx, "failed to send chat notifications",
				slog.Int64("chatID", chatID),
				slog.Any("error", err))
		}
	}
}

// sendChatNotifications отправляет все уведомления в один чат.
func (s *Stats) sendChatNotifications(
	ctx context.Context,
	chatID int64,
	notifies []models.Notify,
) error {
	logger := slog.Default()

	// Создаем чат для отправки
	chat := &telebot.Chat{ID: chatID}

	// Собираем данные для каждого формата
	var pngImages []*bytes.Buffer

	xlsxData := make(map[string][][]string)
	groupName := ""

	var csvData []*bytes.Buffer

	var textMessages []string

	for _, notify := range notifies {
		// Получаем данные из Metabase
		groupName, _ = notify.GetGroupTitle()
		data, err := s.getMetabaseData(ctx, notify)
		if err != nil {
			logger.ErrorContext(ctx, "failed to get metabase data",
				slog.String("name", notify.Name),
				slog.String("query", notify.CardUUID),
				slog.Any("error", err))

			continue
		}

		// Обрабатываем каждый формат
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
				textMsg, err := s.formatTextMessage(notify, data)
				if err != nil {
					logger.ErrorContext(ctx, "failed to format text message",
						slog.String("name", notify.Name),
						slog.Any("error", err))

					continue
				}

				textMessages = append(textMessages, textMsg)

			case models.NotifyFormatCsv:
				csv := s.formatDataAsCSV(data)

				csvData = append(csvData, csv)

			}
		}
	}

	return s.sendData(ctx, xlsxData, csvData, textMessages, pngImages, groupName, chat)
}

func (s *Stats) sendData(
	ctx context.Context,
	xlsxData map[string][][]string,
	csvData []*bytes.Buffer,
	textMessages []string,
	pngImages []*bytes.Buffer,
	title string,
	chat *telebot.Chat,
) error {
	logger := slog.Default()
	if len(pngImages) > 0 {
		err := s.message.SendMedia(chat, pngImages)
		if err != nil {
			logger.ErrorContext(ctx, "failed to send png caption", slog.Any("error", err))
		}
	}

	xlsxBook, err := xlsx.CreateXlsxBook(xlsxData)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create xlsx book", slog.Any("error", err))

		return err
	}

	err = s.message.SendDocument(chat, xlsxBook, title+".xlsx")
	if err != nil {
		logger.ErrorContext(ctx, "failed to send xlsx file", slog.Any("error", err))

		return err
	}

	// Отправляем текстовые сообщения
	for _, textMsg := range textMessages {
		err := s.message.Send(chat, textMsg, &telebot.SendOptions{
			ParseMode: telebot.ModeMarkdown,
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to send text message", slog.Any("error", err))
		}
	}

	if len(csvData) > 0 {
		err := s.message.SendDocument(chat, csvData[0], title+".csv")
		if err != nil {
			logger.ErrorContext(ctx, "failed to send csv file", slog.Any("error", err))
		}
	}

	return nil
}

// formatTextMessage форматирует текстовое сообщение.
func (s *Stats) formatTextMessage(notify models.Notify, data [][]string) (string, error) {
	if notify.TemplateText != nil {
		// Получаем JSON данные для шаблона
		jsonData, err := s.metabase.GetDataMap(context.Background(), notify.CardUUID)
		if err != nil {
			return "", err
		}

		// Заполняем шаблон данными из Metabase
		filledTemplate := s.fillTemplateWithData(*notify.TemplateText, jsonData)

		return filledTemplate, nil
	} else {
		// Форматируем данные как таблицу
		return s.formatDataAsText(data), nil
	}
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
