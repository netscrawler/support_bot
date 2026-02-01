package stdlib_test

import (
	"errors"
	"testing"

	"support_bot/internal/delivery/smtp"
	models "support_bot/internal/models/report"
	"support_bot/internal/plugin/stdlib"
	pmock "support_bot/internal/plugin/stdlib/mock"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestDeliverySend(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		mockSender := pmock.NewMockSender(t)

		mockSender.
			On("Send", mock.Anything, mock.MatchedBy(func(targets []models.Targeted) bool {
				return len(targets) == 1
			}), mock.MatchedBy(func(data []models.ReportData) bool {
				return len(data) == 1
			})).
			Return(nil).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{
			DeliveryPlugin: stdlib.NewDelivery(mockSender, nil, nil, nil),
		}
		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.Send(
			{
				{
					kind = "telegram",
					chat_id = 123456,
					thread_id = 1
				}
			},
			{
				{
					kind = 0,
					msg = "Test message",
					parse = "HTML"
				}
			}
		)
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LNil, luaErr)

		luaResult := L.GetGlobal("result")
		require.Equal(t, true, lua.LVAsBool(luaResult))

		mockSender.AssertExpectations(t)
	})

	t.Run("error case", func(t *testing.T) {
		t.Parallel()
		mockSender := pmock.NewMockSender(t)

		wantErr := errors.New("send failed")

		mockSender.
			On("Send", mock.Anything, mock.Anything, mock.Anything).
			Return(wantErr).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{
			DeliveryPlugin: stdlib.NewDelivery(mockSender, nil, nil, nil),
		}
		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.Send(
			{
				{
					kind = "email",
					dest = {"test@example.com"},
					copy = {},
					subject = "Test",
					body = "Body"
				}
			},
			{
				{
					kind = 0,
					msg = "Test message"
				}
			}
		)
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LString("send failed"), luaErr)

		luaResult := L.GetGlobal("result")
		require.Equal(t, false, lua.LVAsBool(luaResult))

		mockSender.AssertExpectations(t)
	})
}

func TestDeliverySendTelegram(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		mockTgSender := &pmock.MockTelegramChatSender{}

		mockTgSender.
			On("Send", mock.Anything, mock.MatchedBy(func(chat models.TargetTelegramChat) bool {
				return chat.ChatID == 123456 && chat.ThreadID == 1
			}), mock.Anything).
			Return(nil).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{
			DeliveryPlugin: stdlib.NewDelivery(nil, mockTgSender, nil, nil),
		}
		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.SendTelegram(
			123456,
			1,
			{
				{
					kind = 0,
					msg = "Test message",
					parse = "HTML"
				}
			}
		)
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LNil, luaErr)

		luaResult := L.GetGlobal("result")
		require.Equal(t, true, lua.LVAsBool(luaResult))

		mockTgSender.AssertExpectations(t)
	})

	t.Run("error case", func(t *testing.T) {
		t.Parallel()
		mockTgSender := &pmock.MockTelegramChatSender{}

		wantErr := errors.New("telegram send failed")

		mockTgSender.
			On("Send", mock.Anything, mock.Anything, mock.Anything).
			Return(wantErr).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{
			DeliveryPlugin: stdlib.NewDelivery(nil, mockTgSender, nil, nil),
		}
		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.SendTelegram(
			123456,
			0,
			{
				{
					kind = 0,
					msg = "Test message"
				}
			}
		)
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LString("telegram send failed"), luaErr)

		luaResult := L.GetGlobal("result")
		require.Equal(t, false, lua.LVAsBool(luaResult))

		mockTgSender.AssertExpectations(t)
	})
}

func TestDeliverySendFileServer(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		mockFsUploader := &pmock.MockFileUploader{}

		mockFsUploader.
			On("Upload", mock.Anything, "/remote/path/file.csv", mock.Anything).
			Return(nil).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{
			DeliveryPlugin: stdlib.NewDelivery(nil, nil, mockFsUploader, nil),
		}
		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.SendFileServer(
			"/remote/path/file.csv",
			{
				{
					kind = 0,
					msg = "Test data"
				}
			}
		)
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LNil, luaErr)

		luaResult := L.GetGlobal("result")
		require.Equal(t, true, lua.LVAsBool(luaResult))

		mockFsUploader.AssertExpectations(t)
	})

	t.Run("error case", func(t *testing.T) {
		t.Parallel()
		mockFsUploader := &pmock.MockFileUploader{}

		wantErr := errors.New("upload failed")

		mockFsUploader.
			On("Upload", mock.Anything, mock.Anything, mock.Anything).
			Return(wantErr).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{
			DeliveryPlugin: stdlib.NewDelivery(nil, nil, mockFsUploader, nil),
		}
		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.SendFileServer(
			"/remote/path/file.csv",
			{
				{
					kind = 0,
					msg = "Test data"
				}
			}
		)
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LString("upload failed"), luaErr)

		luaResult := L.GetGlobal("result")
		require.Equal(t, false, lua.LVAsBool(luaResult))

		mockFsUploader.AssertExpectations(t)
	})
}

func TestDeliverySendEmail(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		mockSMTPSender := pmock.NewMockSMTPSender(t)

		mockSMTPSender.
			On("Send", mock.Anything, mock.MatchedBy(func(mail smtp.Mail) bool {
				return len(mail.Recipients) == 1 &&
					mail.Recipients[0] == "test@example.com" &&
					mail.Subject == "Test Subject" &&
					mail.Body == "Test Body"
			})).
			Return(nil).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{
			DeliveryPlugin: stdlib.NewDelivery(nil, nil, nil, mockSMTPSender),
		}
		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.SendEmail({
			recipients = {"test@example.com"},
			copy = {},
			subject = "Test Subject",
			body = "Test Body",
			attachments = {}
		})
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LNil, luaErr)

		luaResult := L.GetGlobal("result")
		require.Equal(t, true, lua.LVAsBool(luaResult))

		mockSMTPSender.AssertExpectations(t)
	})

	t.Run("with attachments", func(t *testing.T) {
		t.Parallel()
		mockSMTPSender := pmock.NewMockSMTPSender(t)

		mockSMTPSender.
			On("Send", mock.Anything, mock.MatchedBy(func(mail smtp.Mail) bool {
				return mail.Attachments.Len() == 1
			})).
			Return(nil).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{
			DeliveryPlugin: stdlib.NewDelivery(nil, nil, nil, mockSMTPSender),
		}
		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.SendEmail({
			recipients = {"test@example.com"},
			copy = {"cc@example.com"},
			subject = "Test with attachment",
			body = "Body",
			attachments = {
				{
					name = "file.txt",
					content = "file content"
				}
			}
		})
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LNil, luaErr)

		luaResult := L.GetGlobal("result")
		require.Equal(t, true, lua.LVAsBool(luaResult))

		mockSMTPSender.AssertExpectations(t)
	})

	t.Run("error case", func(t *testing.T) {
		t.Parallel()
		mockSMTPSender := pmock.NewMockSMTPSender(t)

		wantErr := errors.New("smtp send failed")

		mockSMTPSender.
			On("Send", mock.Anything, mock.Anything).
			Return(wantErr).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{
			DeliveryPlugin: stdlib.NewDelivery(nil, nil, nil, mockSMTPSender),
		}
		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.SendEmail({
			recipients = {"test@example.com"},
			copy = {},
			subject = "Test",
			body = "Body",
			attachments = {}
		})
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LString("smtp send failed"), luaErr)

		luaResult := L.GetGlobal("result")
		require.Equal(t, false, lua.LVAsBool(luaResult))

		mockSMTPSender.AssertExpectations(t)
	})
}
