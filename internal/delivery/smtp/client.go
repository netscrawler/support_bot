package smtp

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"path/filepath"
	"strings"
	"time"
)

type SMTPSender struct {
	cfg SMTPConfig
	log *slog.Logger
}

func New(cfg SMTPConfig, log *slog.Logger) *SMTPSender {
	l := log.With(slog.Any("module", "smtp_sender"))

	return &SMTPSender{
		cfg: cfg,
		log: l,
	}
}

func (s *SMTPSender) Send(ctx context.Context, mail Mail) error {
	auth := smtp.PlainAuth("", s.cfg.Email, s.cfg.Password, s.cfg.Host)

	message := s.buildMessage(mail)

	recipients := append(mail.Recipients, mail.Copy...)
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)

	//nolint:gosec //not need
	tlsConfig := &tls.Config{
		ServerName: s.cfg.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		s.log.Error("failed to connect to SMTP server", slog.Any("error", err))

		return fmt.Errorf("connect to SMTP: %w", err)
	}

	defer func() {
		err := conn.Close()
		if err != nil {
			s.log.ErrorContext(ctx, "unable close connection correctly", slog.Any("error", err))
		}
	}()

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		s.log.Error("failed to create SMTP client", slog.Any("error", err))

		return fmt.Errorf("create SMTP client: %w", err)
	}

	defer func() {
		err := client.Quit()
		if err != nil {
			s.log.ErrorContext(ctx, "unable quit from client correctly", slog.Any("error", err))
		}
	}()

	if err = client.Auth(auth); err != nil {
		s.log.Error("failed to authenticate", slog.Any("error", err))

		return fmt.Errorf("authenticate: %w", err)
	}

	if err = client.Mail(s.cfg.Email); err != nil {
		s.log.Error("failed to set sender", slog.Any("error", err))

		return fmt.Errorf("set sender: %w", err)
	}

	for _, recipient := range recipients {
		err = client.Rcpt(recipient)
		if err != nil {
			s.log.Error(
				"failed to add recipient",
				slog.String("recipient", recipient),
				slog.Any("error", err),
			)

			return fmt.Errorf("add recipient %s: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		s.log.Error("failed to get data writer", slog.Any("error", err))

		return fmt.Errorf("get data writer: %w", err)
	}

	if _, err = w.Write(message); err != nil {
		s.log.Error("failed to write message", slog.Any("error", err))

		return fmt.Errorf("write message: %w", err)
	}

	if err = w.Close(); err != nil {
		s.log.Error("failed to close writer", slog.Any("error", err))

		return fmt.Errorf("close writer: %w", err)
	}

	s.log.Info("email sent successfully", slog.Int("recipients", len(recipients)))

	return nil
}

func (s *SMTPSender) buildMessage(mail Mail) []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "From: %s <%s>\r\n", s.cfg.Email, s.cfg.Email)

	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(mail.Recipients, ", "))

	if len(mail.Copy) > 0 {
		fmt.Fprintf(&buf, "Cc: %s\r\n", strings.Join(mail.Copy, ", "))
	}

	encodedSubject := mime.QEncoding.Encode("utf-8", mail.Subject)
	fmt.Fprintf(&buf, "Subject: %s\r\n", encodedSubject)

	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(
		&buf,
		"Message-ID: <%d.%s@%s>\r\n",
		time.Now().UnixNano(),
		randomBoundary(),
		s.cfg.Host,
	)
	buf.WriteString("MIME-Version: 1.0\r\n")

	boundary := "boundary-" + randomBoundary()

	if mail.Attachments.Len() > 0 {
		fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary)

		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		buf.WriteString(mail.Body)
		buf.WriteString("\r\n\r\n")

		for file, name := range mail.Attachments.Data() {
			s.writeAttachment(&buf, boundary, file, name)
		}

		fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	} else {
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		buf.WriteString(mail.Body)
		buf.WriteString("\r\n")
	}

	return buf.Bytes()
}

func (s *SMTPSender) writeAttachment(
	buf *bytes.Buffer,
	boundary string,
	file *bytes.Buffer,
	name string,
) {
	fmt.Fprintf(buf, "--%s\r\n", boundary)

	ext := filepath.Ext(name)

	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	fmt.Fprintf(buf, "Content-Type: %s; name=\"%s\"\r\n", mimeType, name)
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	fmt.Fprintf(buf, "Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", name)

	encoded := base64.StdEncoding.EncodeToString(file.Bytes())

	const maxLineLen = 76

	totalLen := len(encoded)

	for i := 0; i < totalLen; i += maxLineLen {
		end := min(i+maxLineLen, totalLen)
		buf.WriteString(encoded[i:end])
		buf.WriteString("\r\n")
	}

	buf.WriteString("\r\n")
}

func randomBoundary() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)

	return hex.EncodeToString(b)
}
