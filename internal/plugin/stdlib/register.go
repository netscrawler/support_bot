package stdlib

import lua "github.com/yuin/gopher-lua"

type STD struct {
	*CollectPlugin
	*EvaluatorPlugin
	*ExporterPlugin
	*DeliveryPlugin
}

func NewSTD(
	collector Collector,
	evaluator Evaluator,
	exporter ExportFunc,
	sender Sender,
	tg TelegramChatSender,
	fs FileUploader,
	smtp SMTPSender,
) *STD {
	return &STD{
		CollectPlugin:   NewCollector(collector),
		EvaluatorPlugin: NewEvaluator(evaluator),
		ExporterPlugin:  NewExporter(exporter),
		DeliveryPlugin:  NewDelivery(sender, tg, fs, smtp),
	}
}

func (s *STD) Register(L *lua.LState) {
	stdlib := L.NewTable()
	L.SetField(stdlib, "Collect", L.NewFunction(s.luaCollect))
	L.SetField(stdlib, "Evaluate", L.NewFunction(s.luaEvaluate))
	L.SetField(stdlib, "Export", L.NewFunction(s.luaExport))
	L.SetField(stdlib, "Send", L.NewFunction(s.luaSend))
	L.SetField(stdlib, "SendTelegram", L.NewFunction(s.luaSendTelegram))
	L.SetField(stdlib, "SendFileServer", L.NewFunction(s.luaSendFileServer))
	L.SetField(stdlib, "SendEmail", L.NewFunction(s.luaSendEmail))
	L.SetField(stdlib, "ExecuteTemplate", L.NewFunction(luaExecuteTemplate))
	L.SetGlobal("stdlib", stdlib)
}
