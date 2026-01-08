package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"

	"support_bot/internal/exporter/text"
	"support_bot/internal/models"
	"support_bot/internal/pkg/png"

	"github.com/robfig/cron/v3"
)

type ReportGetter interface {
	GetAllActive(ctx context.Context) ([]models.Report, error)
}

type MetabaseDataGetter interface {
	GetDataMatrix(ctx context.Context, cardUUID string) ([][]string, error)
	GetDataMap(ctx context.Context, cardUUID string) ([]map[string]any, error)
	GetDataIter(ctx context.Context, cardUUID string) (iter.Seq[map[string]any], error)
}

type Sender interface {
	Send(meta models.Targeted, data models.ReportData) error
}

type Report struct {
	report   ReportGetter
	sender   Sender
	metabase MetabaseDataGetter
	cron     *cron.Cron
	l        *slog.Logger
}

func New(q ReportGetter, snd Sender, mb MetabaseDataGetter) *Report {
	return &Report{
		report:   q,
		sender:   snd,
		metabase: mb,
		cron:     cron.New(),
		l:        slog.Default(),
	}
}

type CronJobs struct {
	Total    int
	Success  int
	Unsucess map[string]error
}

// Start запускает крон-задачи для всех активных уведомлений.
func (r *Report) Start(ctx context.Context) (CronJobs, error) {
	c := CronJobs{
		Total:    0,
		Success:  0,
		Unsucess: make(map[string]error),
	}
	//
	// Останавливаем предыдущие задачи
	r.cron.Stop()

	// Создаем новый планировщик
	r.cron = cron.New()

	groupedReports, err := r.getActiveReport(ctx)
	if err != nil {
		return CronJobs{}, err
	}

	for groupID, reportGroups := range groupedReports {
		c.Total += len(reportGroups)

		_, err := r.cron.AddFunc(string(reportGroups[0].Cron), func() {
			r.processGroup(reportGroups)
		})
		if err != nil {
			c.Unsucess[groupID] = err
			r.l.ErrorContext(ctx, "failed to add cron job for group",
				slog.String("groupID", groupID),
				slog.Any("error", err))

			continue
		}

		c.Success += len(reportGroups)

		r.l.InfoContext(ctx, "added cron job for group",
			slog.String("groupID", groupID),
			slog.Int("notifyCount", len(reportGroups)))
	}

	// Запускаем планировщик
	r.cron.Start()

	r.l.InfoContext(ctx, "started cron scheduler",
		slog.Int("activeJobs", len(r.cron.Entries())))

	return c, nil
}

// Stop останавливает все крон-задачи.
func (r *Report) Stop() {
	if r.cron != nil {
		slog.Info("stopping crong jobs")
		r.cron.Stop()
	}
}

func (r *Report) getActiveReport(ctx context.Context) (map[string][]models.Report, error) {
	n, err := r.report.GetAllActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w : (%w)", models.ErrInternal, err)
	}

	groupedReports := make(map[string][]models.Report)

	for _, notify := range n {
		groupID := notify.GroupID
		if groupID == "" {
			groupID = notify.Name
		}

		groupedReports[groupID] = append(groupedReports[groupID], notify)
	}

	return groupedReports, nil
}

func (r *Report) processGroup(reports []models.Report) {
	group, title, target := r.collectResults(reports)

	send, err := mergeGroup(group, title)
	if err != nil {
		r.l.Error("failed to merge reports", slog.Any("error", err))

		return
	}

	r.sendGroup(target, send)
}

// обрабатываем все отчёты, получаем список результатов.
func (r *Report) collectResults(
	reports []models.Report,
) ([]models.NotificationResult, string, models.Targeted) {
	group := make([]models.NotificationResult, 0, len(reports))

	var (
		title  string
		target models.Targeted
	)

	for _, rpt := range reports {
		target = rpt.Target
		title = rpt.GroupTitle

		if rpt.CardUUID == nil && len(rpt.CardUUID) == 0 {
			continue
		}

		res, err := r.process(rpt)
		if err != nil {
			r.l.Error("failed to process report", slog.Any("error", err))

			continue
		}

		group = append(group, res)
	}

	return group, title, target
}

// отдельная функция для отправки.
func (r *Report) sendGroup(target models.Targeted, results []models.ReportData) {
	for _, s := range results {
		err := r.sender.Send(target, s)
		if err != nil {
			r.l.Error("failed send report", slog.Any("error", err))
		}
	}
}

func (r *Report) process(report models.Report) (models.NotificationResult, error) {
	var res models.NotificationResult

	res.Title = report.Title

	dataMatrix, dataMap, err := r.exportData(report.CardUUID, report.ReportFormat)
	if err != nil {
		return res, err
	}

	for _, t := range report.ReportFormat {
		switch t {
		case models.NotifyFormatCsv:
			res.CSV = models.NewFileData(writeCsv(dataMatrix), report.Title+".csv")
		case models.NotifyFormatPng:
			var img *bytes.Buffer

			img, err = png.CreateImageFromMatrix(dataMatrix, report.Title, report.Title)
			if err != nil {
			}

			res.Image = models.NewImageData(img, report.Title+".png")

		case models.NotifyFormatXlsx:
			xl := make(map[string][][]string)
			xl[report.Title] = dataMatrix
			res.XLSX = &xl
		case models.NotifyFormatText:
			if report.TemplateText != nil {
				buf, err := text.New(dataMap, *report.TemplateText).Export()
				if err != nil {
					if dataMap == nil {
						return res, errors.New("nil data map")
					}

					return res, err
				}

				txt := buf.Msg
				if txt != "" {
					res.Text = &txt
				}
			}

		case models.NotifyFormatHTML:
			// res.HTML, err = templateHTML("sjfk", make([]map[string]any))

		default:
		}
	}

	return res, nil
}

func (r *Report) exportData(
	cardUUID []string,
	format []models.ReportFormat,
) ([][]string, []map[string]any, error) {
	var rErr error

	needMAP, needMatrix := false, false

	var (
		matrix  [][]string
		dataMap []map[string]any
	)

	for _, t := range format {
		switch t {
		case models.NotifyFormatCsv, models.NotifyFormatXlsx, models.NotifyFormatPng:
			needMatrix = true
		case models.NotifyFormatText, models.NotifyFormatHTML:
			needMAP = true
		default:
			rErr = errors.Join(rErr, fmt.Errorf("unsupported format %s", t))
		}
	}

	for _, cardUUID := range cardUUID {
		if needMAP {
			dm, err := r.metabase.GetDataMap(context.Background(), cardUUID)
			if err != nil {
				rErr = errors.Join(rErr, err)
			} else {
				dataMap = append(dataMap, dm...)
			}
		}

		if needMatrix {
			m, err := r.metabase.GetDataMatrix(context.Background(), cardUUID)
			if err != nil {
				rErr = errors.Join(rErr, err)
			} else {
				// просто склеиваем строки матрицы
				matrix = append(matrix, m...)
			}
		}
	}

	return matrix, dataMap, rErr
}

func mergeGroup(gr []models.NotificationResult, title string) ([]models.ReportData, error) {
	xls := make(map[string][][]string)
	imgs := models.NewEmptyImageData()
	files := models.NewEmptyFileData()
	send := []models.ReportData{}

	for _, r := range gr {
		if r.XLSX != nil {
			maps.Copy(xls, *r.XLSX)
		}

		if r.Text != nil && *r.Text != "" {
			send = append(send, models.NewTextData(*r.Text))
		}

		if r.Image != nil {
			imgs.ExtendIter(r.Image.Data())
		}

		if r.CSV != nil {
			files.ExtendIter(r.CSV.Data())
		}
	}

	// tit, _ := templatex.RenderText(title, nil)

	if len(xls) > 0 {
		//xlsxF, err := xlsx.CreateXlsxBook(xls)
		//if err == nil {
		//files.Extend(xlsxF, tit+".xlsx")
		//}
	}

	if imgs.Entry > 0 {
		send = append(send, imgs)
	}

	if files.Entry > 0 {
		send = append(send, files)
	}

	return send, nil
}

func writeCsv(data [][]string) *bytes.Buffer {
	if len(data) == 0 {
		return nil
	}

	var buf bytes.Buffer

	r := csv.NewWriter(&buf)
	r.WriteAll(data)

	return &buf
}
