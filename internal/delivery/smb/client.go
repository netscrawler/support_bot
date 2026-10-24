package smb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"

	"support_bot/internal/models"
	"support_bot/internal/pkg/retry"

	"github.com/hirochachacha/go-smb2"
)

// share — то, что нужно от *smb2.Share; позволяет подменять шару в тестах.
type share interface {
	WriteFile(name string, data []byte, perm os.FileMode) error
	Stat(name string) (os.FileInfo, error)
	Umount() error
}

type SMB struct {
	cfg    Config
	delays retry.Delays

	mu      sync.RWMutex // защищает conn, session, fs
	conn    net.Conn
	session *smb2.Session
	fs      share

	cancel context.CancelFunc

	log *slog.Logger
}

func New(
	ctx context.Context,
	cfg Config,
	delays retry.Delays,
	log *slog.Logger,
) (*SMB, error) {
	l := log.With(slog.Any("module", "samba_sender"))
	r := &SMB{
		cfg:    cfg,
		delays: delays,
		log:    l,
	}

	if !cfg.Active {
		return r, nil
	}

	err := r.connect(ctx)
	if err != nil {
		l.ErrorContext(
			ctx,
			"Error while connect to share",
			slog.Any("error", err),
		)

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

	if smb.cancel != nil {
		smb.cancel()
	}

	smb.mu.Lock()
	defer smb.mu.Unlock()

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
		return errors.New("smb conn inactive from config")
	}

	remotePath := filepath.Join(remote, data.FileName)
	raw := data.Data.Bytes()

	return retry.Do(
		ctx,
		smb.delays,
		smb.log,
		"smb.Upload",
		func(ctx context.Context) error {
			return smb.write(ctx, remotePath, data.Name(), raw)
		},
	)
}

func (smb *SMB) write(
	ctx context.Context,
	remotePath, name string,
	raw []byte,
) error {
	smb.mu.RLock()
	defer smb.mu.RUnlock()

	if smb.fs == nil {
		return errors.New("smb not connected")
	}

	err := smb.fs.WriteFile(remotePath, raw, 0o644)
	if err != nil {
		smb.log.ErrorContext(
			ctx,
			"failed write remote file",
			slog.Any("file", name),
			slog.Any("error", err),
		)

		return fmt.Errorf(
			"failed to write to remote file %s: %w",
			remotePath,
			err,
		)
	}

	return nil
}
