package text_test

// import (
// 	"fmt"
// 	"testing"
// 	"time"
//
// 	"support_bot/internal/pkg/templatex"
//
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )
//
// func TestRenderText(t *testing.T) {
// 	t.Parallel()
// 	t.Run("SimpleText", func(t *testing.T) {
// 		t.Parallel()
//
// 		got, err := templatex.RenderText("Hello, World!", nil)
// 		require.NoError(t, err)
// 		assert.Equal(t, "Hello, World!", got)
// 	})
//
// 	t.Run("Variable", func(t *testing.T) {
// 		t.Parallel()
//
// 		data := map[string]string{"Name": "Alice"}
// 		got, err := templatex.RenderText("Hello, {{.Name}}!", data)
// 		require.NoError(t, err)
// 		assert.Equal(t, "Hello, Alice!", got)
// 	})
//
// 	t.Run("UpperFunction", func(t *testing.T) {
// 		t.Parallel()
//
// 		data := map[string]string{"Name": "alice"}
// 		got, err := templatex.RenderText("{{upper .Name}}", data)
// 		require.NoError(t, err)
// 		assert.Equal(t, "ALICE", got)
// 	})
//
// 	t.Run("LowerFunction", func(t *testing.T) {
// 		t.Parallel()
//
// 		data := map[string]string{"Name": "ALICE"}
// 		got, err := templatex.RenderText("{{lower .Name}}", data)
// 		require.NoError(t, err)
// 		assert.Equal(t, "alice", got)
// 	})
//
// 	t.Run("AddDays", func(t *testing.T) {
// 		t.Parallel()
//
// 		now := time.Date(2025, time.September, 24, 0, 0, 0, 0, time.UTC)
// 		data := map[string]time.Time{"Now": now}
// 		got, err := templatex.RenderText("{{(addDays .Now 3).Format \"2006-01-02\"}}", data)
// 		require.NoError(t, err)
// 		assert.Equal(t, "2025-09-27", got)
// 	})
//
// 	t.Run("SubDays", func(t *testing.T) {
// 		t.Parallel()
//
// 		now := time.Date(2025, time.September, 24, 0, 0, 0, 0, time.UTC)
// 		data := map[string]time.Time{"Now": now}
// 		got, err := templatex.RenderText("{{(subDays .Now 2).Format \"2006-01-02\"}}", data)
// 		require.NoError(t, err)
// 		assert.Equal(t, "2025-09-22", got)
// 	})
//
// 	t.Run("DiffDays", func(t *testing.T) {
// 		t.Parallel()
//
// 		data := map[string]time.Time{
// 			"A": time.Date(2025, time.September, 24, 0, 0, 0, 0, time.UTC),
// 			"B": time.Date(2025, time.September, 20, 0, 0, 0, 0, time.UTC),
// 		}
// 		got, err := templatex.RenderText("{{diffDays .A .B}}", data)
// 		require.NoError(t, err)
// 		assert.Equal(t, "4", got)
// 	})
//
// 	t.Run("Escape", func(t *testing.T) {
// 		t.Parallel()
//
// 		data := map[string]string{"Text": "_*[]()~`>#+-=|{}.!"}
// 		got, err := templatex.RenderText("{{escape .Text}}", data)
// 		require.NoError(t, err)
// 		assert.Equal(t, `\_\*\[\]\(\)\~\`+"`"+`\>\#\+\-\=\|\{\}\.\!`, got)
// 	})
//
// 	t.Run("InvalidTemplate", func(t *testing.T) {
// 		t.Parallel()
//
// 		_, err := templatex.RenderText("Hello, {{.Name", map[string]string{"Name": "Alice"})
// 		assert.Error(t, err)
// 	})
// }
//
// func TestRenderText_SliceOfMapMultiple(t *testing.T) {
// 	t.Parallel()
// 	t.Run("StringsAndNumbers", func(t *testing.T) {
// 		t.Parallel()
//
// 		data := []map[string]any{
// 			{"Name": "Alice", "Score": 95},
// 			{"Name": "Bob", "Score": 88},
// 		}
//
// 		tmpl := `{{range .}}{{.Name}} scored {{.Score}}
// {{end}}`
//
// 		got, err := templatex.RenderText(tmpl, data)
// 		require.NoError(t, err)
//
// 		want := `Alice scored 95
// Bob scored 88
// `
// 		assert.Equal(t, want, got)
// 	})
//
// 	t.Run("Booleans", func(t *testing.T) {
// 		t.Parallel()
//
// 		data := []map[string]any{
// 			{"Name": "Alice", "Active": true},
// 			{"Name": "Bob", "Active": false},
// 		}
//
// 		tmpl := `{{range .}}{{.Name}} active: {{.Active}}
// {{end}}`
//
// 		got, err := templatex.RenderText(tmpl, data)
// 		require.NoError(t, err)
//
// 		want := `Alice active: true
// Bob active: false
// `
// 		assert.Equal(t, want, got)
// 	})
//
// 	t.Run("Dates", func(t *testing.T) {
// 		t.Parallel()
//
// 		data := []map[string]any{
// 			{"Event": "Launch", "Date": time.Date(2025, time.September, 24, 14, 0, 0, 0, time.UTC)},
// 			{"Event": "Review", "Date": time.Date(2025, time.October, 1, 9, 0, 0, 0, time.UTC)},
// 		}
//
// 		tmpl := `{{range .}}{{.Event}} at {{.Date.Format "2006-01-02 15:04"}}
// {{end}}`
//
// 		got, err := templatex.RenderText(tmpl, data)
// 		require.NoError(t, err)
//
// 		want := `Launch at 2025-09-24 14:00
// Review at 2025-10-01 09:00
// `
// 		assert.Equal(t, want, got)
// 	})
//
// 	t.Run("FunctionsUpperLower", func(t *testing.T) {
// 		t.Parallel()
//
// 		data := []map[string]any{
// 			{"Name": "alice"},
// 			{"Name": "bob"},
// 		}
//
// 		tmpl := `{{range .}}{{upper .Name}} / {{lower .Name}}
// {{end}}`
//
// 		got, err := templatex.RenderText(tmpl, data)
// 		require.NoError(t, err)
//
// 		want := `ALICE / alice
// BOB / bob
// `
// 		assert.Equal(t, want, got)
// 	})
//
// 	t.Run("AddDaysFunction", func(t *testing.T) {
// 		t.Parallel()
//
// 		data := []map[string]any{
// 			{"Name": "Alice", "Date": time.Date(2025, time.September, 24, 0, 0, 0, 0, time.UTC)},
// 			{"Name": "Bob", "Date": time.Date(2025, time.September, 25, 0, 0, 0, 0, time.UTC)},
// 		}
//
// 		tmpl := `{{range .}}{{.Name}} +3 days: {{(addDays .Date 3).Format "2006-01-02"}}
// {{end}}`
//
// 		got, err := templatex.RenderText(tmpl, data)
// 		require.NoError(t, err)
//
// 		want := `Alice +3 days: 2025-09-27
// Bob +3 days: 2025-09-28
// `
// 		assert.Equal(t, want, got)
// 	})
//
// 	t.Run("SingleMapSlice", func(t *testing.T) {
// 		t.Parallel()
//
// 		data := []map[string]any{
// 			{"count": "10"},
// 		}
//
// 		tmpl := `{{(index . 0).count}}`
//
// 		got, err := templatex.RenderText(tmpl, data)
// 		require.NoError(t, err)
//
// 		want := `10`
// 		assert.Equal(t, want, got)
// 	})
// 	t.Run("DateRange", func(t *testing.T) {
// 		t.Parallel()
//
// 		// данные не нужны, так как шаблон использует sprig-функции
// 		tmpl := `{{ subDays now 7 | date "2006-01-02" }} - {{ subDays now 1 | date "2006-01-02" }}`
//
// 		got, err := templatex.RenderText(tmpl, nil)
// 		require.NoError(t, err)
//
// 		// вычисляем ожидаемый результат через Go time, чтобы тест не был "жёстко забит"
// 		now := time.Now()
// 		start := now.AddDate(0, 0, -7).Format(time.DateOnly)
// 		end := now.AddDate(0, 0, -1).Format(time.DateOnly)
// 		want := fmt.Sprintf("%s - %s", start, end)
//
// 		assert.Equal(t, want, got)
// 	})
// }
