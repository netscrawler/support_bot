package smb

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"time"

	"github.com/hirochachacha/go-smb2"
)

func (smb *SMB) startMonitor(ctx context.Context) {
	smb.log.Info("start smb monitor")

	go func() {
		log := smb.log.With(slog.Any("module", "samba_sender_monitor"))

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
	conn, err := net.Dial("tcp", smb.cfg.Adress)
	if err != nil {
		return fmt.Errorf("dial error: %w", err)
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     smb.cfg.User,
			Password: smb.cfg.Password,
			Domain:   smb.cfg.Domain,
		},
	}

	sess, err := d.Dial(conn)
	if err != nil {
		_ = conn.Close()

		return fmt.Errorf("smb session error: %w", err)
	}

	fs, err := sess.Mount(smb.cfg.Share)
	if err != nil {
		_ = sess.Logoff()
		_ = conn.Close()

		return fmt.Errorf("mount error: %w", err)
	}

	smb.conn = conn
	smb.session = sess
	smb.fs = fs

	slog.Info("SMB connected", slog.String("share", smb.cfg.Share))

	return nil
}

func (smb *SMB) reconnectLoop(ctx context.Context) {
	log := smb.log.WithGroup("monitor")

	const (
		initialDelay = time.Second
		maxDelay     = time.Minute
		timeoutEach  = 10 * time.Second
	)

	log.Debug("start reconnect loop")

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
	smb.log.Info("start closing connection")
	smb.cancel()

	if smb.fs != nil {
		err := smb.fs.Umount()
		if err != nil {
			smb.log.Error("error unmount share", slog.Any("error", err))
		}

		smb.fs = nil
	}

	if smb.session != nil {
		err := smb.session.Logoff()
		if err != nil {
			smb.log.Error("error logoff session", slog.Any("error", err))
		}

		smb.session = nil
	}

	if smb.conn != nil {
		err := smb.conn.Close()
		if err != nil {
			smb.log.Error("failed close connection", slog.Any("error", err))
		}

		smb.conn = nil
	}

	smb.log.Info("smb connection closed")
}
