package stdlib

import (
	"bytes"
	"context"
	"time"

	"support_bot/internal/delivery/smtp"
	models "support_bot/internal/models/report"

	lua "github.com/yuin/gopher-lua"
)

type TelegramChatSender interface {
	Send(ctx context.Context, chat models.TargetTelegramChat, data ...models.ReportData) error
}

type FileUploader interface {
	Upload(ctx context.Context, remote string, fileData ...models.ReportData) error
}

type SMTPSender interface {
	Send(ctx context.Context, mail smtp.Mail) error
}

type Sender interface {
	Send(
		ctx context.Context,
		metas []models.Targeted,
		data []models.ReportData,
	) error
}

type DeliveryPlugin struct {
	snd  Sender
	tg   TelegramChatSender
	fs   FileUploader
	smtp SMTPSender
}

func NewDelivery(
	snd Sender,
	tg TelegramChatSender,
	fs FileUploader,
	smtp SMTPSender,
) *DeliveryPlugin {
	return &DeliveryPlugin{
		snd:  snd,
		tg:   tg,
		fs:   fs,
		smtp: smtp,
	}
}

func (p *DeliveryPlugin) luaSend(L *lua.LState) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	luaTargets := L.CheckTable(1)
	luaData := L.CheckTable(2)

	targets, err := targetsFromLua(luaTargets)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	data, err := reportDataFromLua(luaData)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	err = p.snd.Send(ctx, targets, data)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LNil)
	return 2
}

func targetsFromLua(t *lua.LTable) ([]models.Targeted, error) {
	var targets []models.Targeted

	t.ForEach(func(_, v lua.LValue) {
		tbl, ok := v.(*lua.LTable)
		if !ok {
			return
		}

		kind := tbl.RawGetString("kind")
		switch kind.String() {
		case "telegram":
			chatID := tbl.RawGetString("chat_id")
			threadID := tbl.RawGetString("thread_id")

			var tid int
			if threadID.Type() == lua.LTNumber {
				tid = int(lua.LVAsNumber(threadID))
			}

			targets = append(targets, models.NewTargetTelegramChat(
				int64(lua.LVAsNumber(chatID)),
				&tid,
			))

		case "file_server":
			dest := tbl.RawGetString("dest")
			targets = append(targets, models.TargetFileServer{
				Dest: dest.String(),
			})

		case "email":
			destTbl := tbl.RawGetString("dest")
			copyTbl := tbl.RawGetString("copy")
			subject := tbl.RawGetString("subject")
			body := tbl.RawGetString("body")

			var dest, copy []string
			if arr, ok := destTbl.(*lua.LTable); ok {
				arr.ForEach(func(_, val lua.LValue) {
					dest = append(dest, val.String())
				})
			}

			if arr, ok := copyTbl.(*lua.LTable); ok {
				arr.ForEach(func(_, val lua.LValue) {
					copy = append(copy, val.String())
				})
			}

			targets = append(targets, models.TargetEmail{
				Dest:    dest,
				Copy:    copy,
				Subject: subject.String(),
				Body:    body.String(),
			})
		}
	})

	return targets, nil
}

func reportDataFromLua(t *lua.LTable) ([]models.ReportData, error) {
	var data []models.ReportData

	t.ForEach(func(_, v lua.LValue) {
		tbl, ok := v.(*lua.LTable)
		if !ok {
			return
		}

		kind := tbl.RawGetString("kind")
		switch int(lua.LVAsNumber(kind)) {
		case int(models.SendTextKind):
			msg := tbl.RawGetString("msg")
			parse := tbl.RawGetString("parse")

			textData := &models.TextData{
				Msg: msg.String(),
			}

			if parse.Type() == lua.LTString {
				textData.Parse = parse.String()
			}

			data = append(data, textData)
		}
	})

	return data, nil
}

func (p *DeliveryPlugin) luaSendTelegram(L *lua.LState) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	chatID := L.CheckNumber(1)
	threadID := L.OptInt(2, 0)
	luaData := L.CheckTable(3)

	data, err := reportDataFromLua(luaData)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	chat := models.NewTargetTelegramChat(int64(chatID), &threadID)
	err = p.tg.Send(ctx, chat, data...)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LNil)
	return 2
}

func (p *DeliveryPlugin) luaSendFileServer(L *lua.LState) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	remote := L.CheckString(1)
	luaData := L.CheckTable(2)

	data, err := reportDataFromLua(luaData)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	err = p.fs.Upload(ctx, remote, data...)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LNil)
	return 2
}

func (p *DeliveryPlugin) luaSendEmail(L *lua.LState) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	luaMail := L.CheckTable(1)

	mail, err := mailFromLua(luaMail)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	err = p.smtp.Send(ctx, mail)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LNil)
	return 2
}

func mailFromLua(t *lua.LTable) (smtp.Mail, error) {
	recipientsTbl := t.RawGetString("recipients")
	copyTbl := t.RawGetString("copy")
	subject := t.RawGetString("subject")
	body := t.RawGetString("body")
	attachmentsTbl := t.RawGetString("attachments")

	var recipients, copy []string
	if arr, ok := recipientsTbl.(*lua.LTable); ok {
		arr.ForEach(func(_, val lua.LValue) {
			recipients = append(recipients, val.String())
		})
	}

	if arr, ok := copyTbl.(*lua.LTable); ok {
		arr.ForEach(func(_, val lua.LValue) {
			copy = append(copy, val.String())
		})
	}

	fileData := models.NewEmptyFileData()
	if arr, ok := attachmentsTbl.(*lua.LTable); ok {
		arr.ForEach(func(_, val lua.LValue) {
			if attTbl, ok := val.(*lua.LTable); ok {
				name := attTbl.RawGetString("name")
				content := attTbl.RawGetString("content")
				fileData.ExtendWithoutTemplate(
					bytes.NewBuffer([]byte(content.String())),
					name.String(),
				)
			}
		})
	}

	return smtp.Mail{
		Recipients:  recipients,
		Copy:        copy,
		Subject:     subject.String(),
		Body:        body.String(),
		Attachments: *fileData,
	}, nil
}
