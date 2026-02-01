package stdlib

import (
	"context"
	"time"

	models "support_bot/internal/models/report"

	lua "github.com/yuin/gopher-lua"
)

type ExportFunc func(
	ctx context.Context,
	data map[string][]map[string]any,
	exp models.Export,
) (models.ReportData, error)

type ExporterPlugin struct {
	exp ExportFunc
}

func NewExporter(exp ExportFunc) *ExporterPlugin {
	return &ExporterPlugin{exp: exp}
}

func (p *ExporterPlugin) luaExport(L *lua.LState) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	luaData := L.CheckTable(1)
	luaExport := L.CheckTable(2)

	data, err := luaTableToGoData(luaData)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	exp, err := exportFromLua(luaExport)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	result, err := p.exp(ctx, data, exp)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(reportDataToLua(L, result))
	L.Push(lua.LNil)
	return 2
}

func exportFromLua(t *lua.LTable) (models.Export, error) {
	format := t.RawGetString("format")
	fileName := t.RawGetString("file_name")
	orderTbl := t.RawGetString("order")

	var order map[string][]string
	if tbl, ok := orderTbl.(*lua.LTable); ok {
		order = make(map[string][]string)
		tbl.ForEach(func(k, v lua.LValue) {
			key := k.String()
			if arr, ok := v.(*lua.LTable); ok {
				var cols []string
				arr.ForEach(func(_, val lua.LValue) {
					cols = append(cols, val.String())
				})
				order[key] = cols
			}
		})
	}

	exp := models.Export{
		Format: models.ReportFormat(format.String()),
		Order:  order,
	}

	if fileName.Type() == lua.LTString {
		fn := fileName.String()
		exp.FileName = &fn
	}

	templateTbl := t.RawGetString("template")
	if tbl, ok := templateTbl.(*lua.LTable); ok {
		text := tbl.RawGetString("template_text")
		if text.Type() == lua.LTString {
			exp.Template = &models.Template{
				TemplateText: text.String(),
			}
		}
	}

	return exp, nil
}

func reportDataToLua(L *lua.LState, data models.ReportData) *lua.LTable {
	result := L.NewTable()
	result.RawSetString("kind", lua.LNumber(data.Kind()))
	return result
}
