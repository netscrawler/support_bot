package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"support_bot/internal/models"
	mocks "support_bot/internal/service/mock"
)

func TestReport_collectResults(t *testing.T) {
	t.Parallel()
	t.Run("success single report", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().GetDataMatrix(mock.Anything, "uuid").Return([][]string{{"a", "b"}}, nil)

		rpt := models.Report{
			CardUUID:     []string{"uuid"},
			GroupTitle:   "TestGroup",
			Target:       models.NewTargetTelegramChat(1213, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatCsv},
			Title:        "TestReport",
		}

		r := New(nil, nil, mb)
		wantf := writeCsv([][]string{{"a", "b"}})
		want := models.NotificationResult{
			Title: rpt.Title,
			CSV:   models.NewFileData(wantf, "TestReport.csv"),
		}

		got, gotTitle, gotTarget := r.collectResults([]models.Report{rpt})
		require.Len(t, got, 1)
		assert.Equal(t, "TestGroup", gotTitle)
		assert.Equal(t, models.TargetTelegramChatKind, gotTarget.Kind())
		tr, ok := gotTarget.(models.TargetTelegramChat)
		assert.True(t, ok)
		assert.Equal(t, int64(1213), tr.ChatID)
		assert.Equal(t, want, got[0])
	})

	t.Run("success double query report", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().GetDataMatrix(mock.Anything, "uuid").Return([][]string{{"a", "b"}}, nil)
		mb.EXPECT().GetDataMatrix(mock.Anything, "uuid1").Return([][]string{{"b", "c"}}, nil)

		rpt := models.Report{
			CardUUID:     []string{"uuid", "uuid1"},
			GroupTitle:   "TestGroup",
			Target:       models.NewTargetTelegramChat(1213, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatCsv},
			Title:        "TestReport",
		}
		wantf := writeCsv([][]string{{"a", "b"}, {"b", "c"}})
		want := models.NotificationResult{
			Title: rpt.Title,
			CSV:   models.NewFileData(wantf, "TestReport.csv"),
		}

		r := New(nil, nil, mb)

		got, gotTitle, gotTarget := r.collectResults([]models.Report{rpt})
		require.Len(t, got, 1)
		assert.Equal(t, "TestGroup", gotTitle)
		assert.Equal(t, models.TargetTelegramChatKind, gotTarget.Kind())
		tr, ok := gotTarget.(models.TargetTelegramChat)
		assert.True(t, ok)
		assert.Equal(t, int64(1213), tr.ChatID)
		assert.Equal(t, want, got[0])
	})

	t.Run("error from metabase", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().
			GetDataMatrix(mock.Anything, "uuid").
			Return(nil, errors.New("db error"))

		rpt := models.Report{
			CardUUID:     []string{"uuid"},
			GroupTitle:   "ErrGroup",
			Target:       models.NewTargetTelegramChat(42, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatCsv},
			Title:        "ErrReport",
		}

		r := New(nil, nil, mb)
		got, gotTitle, gotTarget := r.collectResults([]models.Report{rpt})
		require.Empty(t, got)
		assert.Equal(t, "ErrGroup", gotTitle)
		assert.Equal(t, models.TargetTelegramChatKind, gotTarget.Kind())
	})

	t.Run("empty reports slice", func(t *testing.T) {
		t.Parallel()

		r := New(nil, nil, nil)

		got, gotTitle, gotTarget := r.collectResults([]models.Report{})
		require.Empty(t, got)
		assert.Empty(t, gotTitle)
		assert.Nil(t, gotTarget)
	})
	t.Run("success double format report", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().
			GetDataMatrix(mock.Anything, "uuid").
			Return([][]string{{"a", "b"}}, nil)
		mb.EXPECT().
			GetDataMap(mock.Anything, "uuid").
			Return([]map[string]any{{"a": "world"}}, nil)

		rpt := models.Report{
			CardUUID:     []string{"uuid"},
			GroupTitle:   "TestGroup",
			Target:       models.NewTargetTelegramChat(1213, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatCsv, models.NotifyFormatText},
			Title:        "TestReport",
			TemplateText: ptr(`HELLO {{index . 0 "a"}}`),
		}

		r := New(nil, nil, mb)
		wantf := writeCsv([][]string{{"a", "b"}})
		want := models.NotificationResult{
			Title: rpt.Title,
			CSV:   models.NewFileData(wantf, "TestReport.csv"),
			Text:  ptr(`HELLO world`),
		}

		got, gotTitle, gotTarget := r.collectResults([]models.Report{rpt})
		require.Len(t, got, 1)
		assert.Equal(t, "TestGroup", gotTitle)
		assert.Equal(t, models.TargetTelegramChatKind, gotTarget.Kind())
		tr, ok := gotTarget.(models.TargetTelegramChat)
		assert.True(t, ok)
		assert.Equal(t, int64(1213), tr.ChatID)
		assert.Equal(t, want, got[0])
	})

	t.Run("success double query text report", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().
			GetDataMap(mock.Anything, "uuid").
			Return([]map[string]any{{"a": "world"}}, nil)
		mb.EXPECT().
			GetDataMap(mock.Anything, "uuid1").
			Return([]map[string]any{{"b": "hallo"}}, nil)

		rpt := models.Report{
			CardUUID:     []string{"uuid", "uuid1"},
			GroupTitle:   "TestGroup",
			Target:       models.NewTargetTelegramChat(1213, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatText},
			Title:        "TestReport",
			TemplateText: ptr(`HELLO {{index . 0 "a"}} {{index . 1 "b"}}`),
		}

		r := New(nil, nil, mb)
		want := models.NotificationResult{Title: rpt.Title, Text: ptr(`HELLO world hallo`)}

		got, gotTitle, gotTarget := r.collectResults([]models.Report{rpt})
		require.Len(t, got, 1)
		assert.Equal(t, "TestGroup", gotTitle)
		assert.Equal(t, models.TargetTelegramChatKind, gotTarget.Kind())
		tr, ok := gotTarget.(models.TargetTelegramChat)
		assert.True(t, ok)
		assert.Equal(t, int64(1213), tr.ChatID)
		assert.Equal(t, want, got[0])
	})
	t.Run("success without query text report", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)

		rpt := models.Report{
			CardUUID:     []string{},
			GroupTitle:   "TestGroup",
			Target:       models.NewTargetTelegramChat(1213, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatText},
			Title:        "TestReport",
			TemplateText: ptr(`HELLO`),
		}

		r := New(nil, nil, mb)
		want := models.NotificationResult{Title: rpt.Title, Text: ptr(`HELLO`)}

		got, gotTitle, gotTarget := r.collectResults([]models.Report{rpt})
		require.Len(t, got, 1)
		assert.Equal(t, "TestGroup", gotTitle)
		assert.Equal(t, models.TargetTelegramChatKind, gotTarget.Kind())
		tr, ok := gotTarget.(models.TargetTelegramChat)
		assert.True(t, ok)
		assert.Equal(t, int64(1213), tr.ChatID)
		assert.Equal(t, want, got[0])
	})
	t.Run("unsupported format", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)

		rpt := models.Report{
			CardUUID:     []string{"uuid"},
			GroupTitle:   "UnsupportedGroup",
			Target:       models.NewTargetTelegramChat(123, nil),
			ReportFormat: []models.ReportFormat{"unknown_format"}, // неподдерживаемый формат
			Title:        "UnsupportedReport",
		}

		r := New(nil, nil, mb)
		got, gotTitle, gotTarget := r.collectResults([]models.Report{rpt})

		require.Empty(t, got)                                            // нет результатов
		assert.Equal(t, "UnsupportedGroup", gotTitle)                    // группа верна
		assert.Equal(t, models.TargetTelegramChatKind, gotTarget.Kind()) // Target корректен
	})

	t.Run("empty map from GetDataMap", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().
			GetDataMap(mock.Anything, "uuid").
			Return([]map[string]any{}, nil) // пустой слайс карт

		rpt := models.Report{
			CardUUID:     []string{"uuid"},
			GroupTitle:   "EmptyMapGroup",
			Target:       models.NewTargetTelegramChat(456, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatText},
			Title:        "EmptyMapReport",
			TemplateText: ptr(`HELLO {{index . 0 "a"}}`), // шаблон с обращением к первой карте
		}

		r := New(nil, nil, mb)
		got, gotTitle, gotTarget := r.collectResults([]models.Report{rpt})

		// Поскольку dataMap пустая, collectResults вернёт nil data map → результат отсутствует
		require.Empty(t, got)                      // нет результатов
		assert.Equal(t, "EmptyMapGroup", gotTitle) // группа корректна
		assert.Equal(t, models.TargetTelegramChatKind, gotTarget.Kind())
		tr, ok := gotTarget.(models.TargetTelegramChat)
		assert.True(t, ok)
		assert.Equal(t, int64(456), tr.ChatID)
	})
}

func TestReport_sendGroup(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		q   ReportGetter
		snd Sender
		mb  MetabaseDataGetter
		// Named input parameters for target function.
		target  models.Targeted
		results []models.Sendable
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(tt.q, tt.snd, tt.mb)
			r.sendGroup(tt.target, tt.results)
		})
	}
}

func TestReport_process(t *testing.T) {
	t.Parallel()

	t.Run("single CSV report", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().
			GetDataMatrix(mock.Anything, "uuid").
			Return([][]string{{"a", "b"}}, nil)

		rpt := models.Report{
			GroupTitle:   "CSVReport",
			CardUUID:     []string{"uuid"},
			Target:       models.NewTargetTelegramChat(123, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatCsv},
			Title:        "CSVReport",
		}

		r := New(nil, nil, mb)
		got, err := r.process(rpt)
		require.NoError(t, err)

		wantf := writeCsv([][]string{{"a", "b"}})
		want := models.NotificationResult{
			Title: rpt.Title,
			CSV:   models.NewFileData(wantf, "CSVReport.csv"),
		}

		assert.Equal(t, want, got)
	})

	t.Run("single Text report", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().
			GetDataMap(mock.Anything, "uuid").
			Return([]map[string]any{{"a": "world"}}, nil)

		rpt := models.Report{
			CardUUID:     []string{"uuid"},
			Target:       models.NewTargetTelegramChat(456, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatText},
			Title:        "TextReport",
			TemplateText: ptr(`HELLO {{index . 0 "a"}}`),
		}

		r := New(nil, nil, mb)
		got, err := r.process(rpt)
		require.NoError(t, err)

		want := models.NotificationResult{
			Title: rpt.Title,
			Text:  ptr("HELLO world"),
		}

		assert.Equal(t, want, got)
	})

	t.Run("double format report CSV+Text", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().GetDataMatrix(mock.Anything, "uuid").
			Return([][]string{{"a", "b"}}, nil)
		mb.EXPECT().GetDataMap(mock.Anything, "uuid").
			Return([]map[string]any{{"a": "world"}}, nil)

		rpt := models.Report{
			CardUUID:     []string{"uuid"},
			Target:       models.NewTargetTelegramChat(789, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatCsv, models.NotifyFormatText},
			Title:        "CSVTextReport",
			TemplateText: ptr(`HELLO {{index . 0 "a"}}`),
		}

		r := New(nil, nil, mb)
		got, err := r.process(rpt)
		require.NoError(t, err)

		wantf := writeCsv([][]string{{"a", "b"}})
		want := models.NotificationResult{
			Title: rpt.Title,
			CSV:   models.NewFileData(wantf, "CSVTextReport.csv"),
			Text:  ptr("HELLO world"),
		}

		assert.Equal(t, want, got)
	})

	t.Run("error from Metabase", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().GetDataMatrix(mock.Anything, "uuid").
			Return(nil, errors.New("db error"))

		rpt := models.Report{
			CardUUID:     []string{"uuid"},
			Target:       models.NewTargetTelegramChat(111, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatCsv},
			Title:        "ErrReport",
		}

		r := New(nil, nil, mb)
		got, err := r.process(rpt)
		require.Error(t, err)
		assert.Equal(t, models.NotificationResult{Title: rpt.Title}, got)
	})

	t.Run("empty map from GetDataMap with noErr", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().GetDataMap(mock.Anything, "uuid").
			Return([]map[string]any{}, nil)

		rpt := models.Report{
			CardUUID:     []string{"uuid"},
			Target:       models.NewTargetTelegramChat(222, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatText},
			Title:        "EmptyMapReport",
			TemplateText: ptr(`HELLO {{index 0 .a}}`),
		}

		r := New(nil, nil, mb)
		got, err := r.process(rpt)
		require.Error(t, err)

		// Поскольку карта пустая, результат пустой
		assert.Equal(t, models.NotificationResult{
			Title: rpt.Title,
		}, got)
	})

	t.Run("empty map from GetDataMap without report", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().GetDataMap(mock.Anything, "uuid").
			Return([]map[string]any{}, nil)

		rpt := models.Report{
			CardUUID:     []string{"uuid"},
			Target:       models.NewTargetTelegramChat(222, nil),
			ReportFormat: []models.ReportFormat{models.NotifyFormatText},
			Title:        "EmptyMapReport",
			TemplateText: ptr(`HELLO`),
		}

		r := New(nil, nil, mb)
		got, err := r.process(rpt)
		require.NoError(t, err)

		// Поскольку карта пустая, результат пустой
		assert.Equal(t, models.NotificationResult{
			Title: rpt.Title,
			Text:  ptr(`HELLO`),
		}, got)
	})
}

func TestReport_exportData(t *testing.T) {
	t.Parallel()

	t.Run("single matrix", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().
			GetDataMatrix(mock.Anything, "uuid").
			Return([][]string{{"a", "b"}}, nil)

		r := New(nil, nil, mb)
		got, got2, err := r.exportData(
			[]string{"uuid"},
			[]models.ReportFormat{models.NotifyFormatCsv},
		)
		require.NoError(t, err)
		assert.Equal(t, [][]string{{"a", "b"}}, got)
		assert.Nil(t, got2)
	})

	t.Run("single map", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().
			GetDataMap(mock.Anything, "uuid").
			Return([]map[string]any{{"a": "world"}}, nil)

		r := New(nil, nil, mb)
		got, got2, err := r.exportData(
			[]string{"uuid"},
			[]models.ReportFormat{models.NotifyFormatText},
		)
		require.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, []map[string]any{{"a": "world"}}, got2)
	})

	t.Run("double query matrix", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().GetDataMatrix(mock.Anything, "uuid").
			Return([][]string{{"a", "b"}}, nil)
		mb.EXPECT().GetDataMatrix(mock.Anything, "uuid1").
			Return([][]string{{"b", "c"}}, nil)

		r := New(nil, nil, mb)
		got, got2, err := r.exportData(
			[]string{"uuid", "uuid1"},
			[]models.ReportFormat{models.NotifyFormatCsv},
		)
		require.NoError(t, err)
		assert.Equal(t, [][]string{{"a", "b"}, {"b", "c"}}, got)
		assert.Nil(t, got2)
	})

	t.Run("double query map", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().GetDataMap(mock.Anything, "uuid").
			Return([]map[string]any{{"a": "world"}}, nil)
		mb.EXPECT().GetDataMap(mock.Anything, "uuid1").
			Return([]map[string]any{{"b": "hallo"}}, nil)

		r := New(nil, nil, mb)
		got, got2, err := r.exportData(
			[]string{"uuid", "uuid1"},
			[]models.ReportFormat{models.NotifyFormatText},
		)
		require.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, []map[string]any{{"a": "world"}, {"b": "hallo"}}, got2)
	})

	t.Run("unsupported format", func(t *testing.T) {
		t.Parallel()

		r := New(nil, nil, nil)
		got, got2, err := r.exportData([]string{"uuid"}, []models.ReportFormat{"unknown_format"})
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Nil(t, got2)
	})

	t.Run("error from Metabase", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().
			GetDataMatrix(mock.Anything, "uuid").
			Return(nil, errors.New("db error"))

		r := New(nil, nil, mb)
		got, got2, err := r.exportData(
			[]string{"uuid"},
			[]models.ReportFormat{models.NotifyFormatCsv},
		)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Nil(t, got2)
	})

	t.Run("empty map from GetDataMap", func(t *testing.T) {
		t.Parallel()
		mb := mocks.NewMockMetabaseDataGetter(t)
		mb.EXPECT().
			GetDataMap(mock.Anything, "uuid").
			Return([]map[string]any{}, nil)

		r := New(nil, nil, mb)
		got, got2, err := r.exportData(
			[]string{"uuid"},
			[]models.ReportFormat{models.NotifyFormatText},
		)
		require.NoError(t, err)
		assert.Nil(t, got)
		assert.Empty(t, got2)
	})
}

func ptr[T any](t T) *T {
	return &t
}
