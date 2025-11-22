package service

import (
	"testing"

	_ "github.com/stretchr/testify/assert"
	"support_bot/internal/models"
)

func TestSenderStrategy_Send(t *testing.T) {
}

func TestSenderStrategy_sendTelegramDataStrategy(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		tg  telegramChatSender
		smb fileUploader
		// Named input parameters for target function.
		target  models.TargetTelegramChat
		data    models.Sendable
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := NewSender(tt.tg, tt.smb)

			gotErr := ss.sendTelegramDataStrategy(tt.target, tt.data)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("sendTelegramDataStrategy() failed: %v", gotErr)
				}

				return
			}

			if tt.wantErr {
				t.Fatal("sendTelegramDataStrategy() succeeded unexpectedly")
			}
		})
	}
}

func TestSenderStrategy_sendSMBDataStrategy(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		tg  telegramChatSender
		smb fileUploader
		// Named input parameters for target function.
		target  models.TargetFileServer
		data    models.Sendable
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := NewSender(tt.tg, tt.smb)

			gotErr := ss.sendSMBDataStrategy(tt.target, tt.data)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("sendSMBDataStrategy() failed: %v", gotErr)
				}

				return
			}

			if tt.wantErr {
				t.Fatal("sendSMBDataStrategy() succeeded unexpectedly")
			}
		})
	}
}
