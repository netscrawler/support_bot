package smb

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	models "support_bot/internal/models/report"

	"github.com/hirochachacha/go-smb2"
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

	if !cfg.Active {
		return r, nil
	}

	err := r.connect()
	if err != nil {
		l.ErrorContext(ctx, "Error while connect to share", slog.Any("error", err))

		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.startMonitor(ctx)

	return r, nil
}

func (smb *SMB) Close() error {
	if !smb.cfg.Active {
		return nil
	}
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

func (smb *SMB) Upload(
	ctx context.Context,
	remote string,
	data models.Data,
) error {
	if !smb.cfg.Active {
		return fmt.Errorf("smb conn inactive from config")
	}

	remotePath := filepath.Join(remote, data.Name)

	f, err := smb.fs.Create(remotePath)
	if err != nil {
		smb.log.ErrorContext(
			ctx,
			"failed create remote file",
			slog.Any("file", data.Name),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to create remote file %s: %w", remotePath, err)

	}

	_, err = f.Write(data.Data.Bytes())

	defer func() {
		err := f.Close()
		if err != nil {
			smb.log.ErrorContext(ctx, "unable close file body", slog.Any("error", err))
		}
	}()

	if err != nil {
		smb.log.ErrorContext(
			ctx,
			"failed write remote file",
			slog.Any("file", data.Name),
			slog.Any("error", err),
		)

		return fmt.Errorf("failed to write to remote file %s: %w", remotePath, err)

	}

	return nil
}
