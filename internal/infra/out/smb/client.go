package smb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"path/filepath"
	"time"

	"support_bot/internal/models"

	"github.com/hirochachacha/go-smb2"
)

type SMB struct {
	address  string
	user     string
	password string
	domain   string
	share    string

	conn    net.Conn
	session *smb2.Session
	fs      *smb2.Share

	cancel context.CancelFunc
}

func New(
	ctx context.Context,
	address, user, password, domain, share string,
) (*SMB, error) {
	r := &SMB{
		address:  address,
		user:     user,
		password: password,
		domain:   domain,
		share:    share,
	}

	if err := r.connect(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.startMonitor(ctx)

	return r, nil
}

func (smb *SMB) Upload(remote string, fileData *models.FileData) error {
	if smb.fs == nil {
		return errors.New("SMB share is not mounted")
	}

	var uploadErr error

	fileData.Data()(func(buf *bytes.Buffer, name string) bool {
		remotePath := filepath.Join(remote, name)

		f, err := smb.fs.Create(remotePath)
		if err != nil {
			uploadErr = fmt.Errorf("failed to create remote file %s: %w", remotePath, err)
			return false // прекращаем итерацию при ошибке
		}

		_, err = f.Write(buf.Bytes())
		f.Close() // закрываем сразу после записи
		if err != nil {
			uploadErr = fmt.Errorf("failed to write to remote file %s: %w", remotePath, err)
			return false
		}

		return true // продолжаем итерацию
	})

	return uploadErr
}

func (smb *SMB) Close() error {
	smb.cancel()
	if smb.fs != nil {
		_ = smb.fs.Umount()
	}
	if smb.session != nil {
		_ = smb.session.Logoff()
	}
	if smb.conn != nil {
		return smb.conn.Close()
	}
	return nil
}

func (smb *SMB) startMonitor(ctx context.Context) {
	go func() {
		log := slog.Default()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("SMB monitor stopped", slog.String("reason", ctx.Err().Error()))
				return
			case <-ticker.C:
				if _, err := smb.fs.Stat("."); err != nil {
					log.Warn("SMB check failed, starting reconnect", slog.Any("error", err))
					smb.reconnectLoop(ctx)
				}
			}
		}
	}()
}

func (smb *SMB) connect() error {
	conn, err := net.Dial("tcp", smb.address)
	if err != nil {
		return fmt.Errorf("dial error: %w", err)
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     smb.user,
			Password: smb.password,
			Domain:   smb.domain,
		},
	}

	sess, err := d.Dial(conn)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smb session error: %w", err)
	}

	fs, err := sess.Mount(smb.share)
	if err != nil {
		_ = sess.Logoff()
		_ = conn.Close()
		return fmt.Errorf("mount error: %w", err)
	}

	smb.conn = conn
	smb.session = sess
	smb.fs = fs

	slog.Info("SMB connected", slog.String("share", smb.share))
	return nil
}

func (smb *SMB) reconnectLoop(ctx context.Context) {
	log := slog.Default()

	const (
		initialDelay = time.Second
		maxDelay     = time.Minute
		timeoutEach  = 10 * time.Second
	)

	delay := initialDelay

	for {
		select {
		case <-ctx.Done():
			log.Info("SMB reconnect loop stopped", slog.String("reason", ctx.Err().Error()))
			return
		default:
			smb.cleanup()

			err := smb.connect()

			if err == nil {
				log.Info("SMB reconnected successfully")
				return
			}

			log.Warn("Reconnect attempt failed", slog.Any("error", err))

			//nolint: gosec
			sleep := delay/2 + time.Duration(rand.Int63n(int64(delay/2)))
			log.Info("Waiting before next retry", slog.Duration("delay", sleep))
			select {
			case <-ctx.Done():
				return
			case <-time.After(sleep):
			}

			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
}

func (smb *SMB) cleanup() {
	if smb.fs != nil {
		_ = smb.fs.Umount()
		smb.fs = nil
	}
	if smb.session != nil {
		_ = smb.session.Logoff()
		smb.session = nil
	}
	if smb.conn != nil {
		_ = smb.conn.Close()
		smb.conn = nil
	}
}
