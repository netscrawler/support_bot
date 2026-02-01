package stdlib_test

import (
	"context"
	"errors"
	"testing"

	models "support_bot/internal/models/report"
	"support_bot/internal/plugin/stdlib"

	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestExporter(t *testing.T) {
	t.Parallel()

	t.Run("happy path - csv export", func(t *testing.T) {
		t.Parallel()

		expectedResult := models.NewTextData("exported data")

		exportFunc := func(ctx context.Context,
			data map[string][]map[string]any,
			exp models.Export,
		) (models.ReportData, error) {
			return expectedResult, nil
		}

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{ExporterPlugin: stdlib.NewExporter(exportFunc)}
		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.Export(
			{
				Sheet1 = {
					{ col1 = "value1", col2 = 100 }
				}
			},
			{
				format = "csv",
				file_name = "report.csv",
				order = {
					Sheet1 = { "col1", "col2" }
				}
			}
		)
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LNil, luaErr)

		luaResult := L.GetGlobal("result")
		require.IsType(t, &lua.LTable{}, luaResult)
	})

	t.Run("export with template", func(t *testing.T) {
		t.Parallel()

		expectedResult := models.NewTextData("formatted text")

		exportFunc := func(ctx context.Context,
			data map[string][]map[string]any,
			exp models.Export,
		) (models.ReportData, error) {
			return expectedResult, nil
		}

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{ExporterPlugin: stdlib.NewExporter(exportFunc)}
		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.Export(
			{
				Data = {
					{ name = "John", age = 30 }
				}
			},
			{
				format = "text",
				template = {
					template_text = "Hello {{.name}}"
				}
			}
		)
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LNil, luaErr)

		luaResult := L.GetGlobal("result")
		require.IsType(t, &lua.LTable{}, luaResult)
	})

	t.Run("error case", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("export failed")

		exportFunc := func(ctx context.Context,
			data map[string][]map[string]any,
			exp models.Export,
		) (models.ReportData, error) {
			return nil, wantErr
		}

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{ExporterPlugin: stdlib.NewExporter(exportFunc)}
		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.Export(
			{
				Data = {
					{ field = "value" }
				}
			},
			{
				format = "xlsx"
			}
		)
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LString("export failed"), luaErr)

		luaResult := L.GetGlobal("result")
		require.Equal(t, lua.LNil, luaResult)
	})
}
