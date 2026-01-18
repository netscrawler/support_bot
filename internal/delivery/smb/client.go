package smb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"path/filepath"

	"github.com/hirochachacha/go-smb2"
	models "support_bot/internal/models/report"
)

type SMB struct {
	cfg SMBConfig

	conn    net.Conn
	session *smb2.Session
	fs      *smb2.Share

	cancel context.CancelFunc

	log *slog.Logger
}

func New(
	ctx context.Context,
	cfg SMBConfig,
	log *slog.Logger,
) (*SMB, error) {
	l := log.With(slog.Any("module", "samba_sender"))
	r := &SMB{
		cfg: cfg,
		log: l,
	}

	err := r.connect()
	if err != nil {
		l.Error("Error while connect to share", slog.Any("error", err))

		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.startMonitor(ctx)

	return r, nil
}

func (smb *SMB) Upload(
	remote string,
	fileData *models.FileData,
	imageData *models.ImageData,
) error {
	if smb.fs == nil {
		smb.log.Error("error upload file", slog.Any("error", "smb share is not mounted"))

		return errors.New("SMB share is not mounted")
	}

	l := smb.log.With("start uploading file", slog.Any("remote_path", remote))

	var uploadErr error

	if fileData != nil {
		fileData.Data()(func(buf *bytes.Buffer, name string) bool {
			remotePath := filepath.Join(remote, name)

			f, err := smb.fs.Create(remotePath)
			if err != nil {
				l.Error("failed create remote file", slog.Any("file", name), slog.Any("error", err))
				uploadErr = fmt.Errorf("failed to create remote file %s: %w", remotePath, err)

				return false
			}

			_, err = f.Write(buf.Bytes())
			f.Close()

			if err != nil {
				l.Error("failed write remote file", slog.Any("file", name), slog.Any("error", err))

				uploadErr = fmt.Errorf("failed to write to remote file %s: %w", remotePath, err)

				return false
			}

			return true
		})
	}

	if imageData == nil {
		return uploadErr
	}

	imageData.Data()(func(buf *bytes.Buffer, name string) bool {
		remotePath := filepath.Join(remote, name)

		f, err := smb.fs.Create(remotePath)
		if err != nil {
			l.Error("failed create remote file", slog.Any("file", name), slog.Any("error", err))
			uploadErr = fmt.Errorf("failed to create remote file %s: %w", remotePath, err)

			return false
		}

		_, err = f.Write(buf.Bytes())
		f.Close()

		if err != nil {
			l.Error("failed write remote file", slog.Any("file", name), slog.Any("error", err))

			uploadErr = fmt.Errorf("failed to write to remote file %s: %w", remotePath, err)

			return false
		}

		return true
	})

	return uploadErr
}

func (smb *SMB) Close() error {
	smb.log.Info("start closing connection")
	smb.cancel()

	if smb.fs != nil {
		err := smb.fs.Umount()
		if err != nil {
			smb.log.Error("error unmount share", slog.Any("error", err))
		}
	}

	if smb.session != nil {
		err := smb.session.Logoff()
		if err != nil {
			smb.log.Error("error logoff session", slog.Any("error", err))
		}
	}

	if smb.conn != nil {
		err := smb.conn.Close()
		if err != nil {
			smb.log.Error("failed close connection", slog.Any("error", err))
		}

		return err
	}

	smb.log.Info("smb connection closed")

	return nil
}
