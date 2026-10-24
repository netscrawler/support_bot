package smtp

import "support_bot/internal/pkg/secret"

type Config struct {
	Host     string        `comment:"Хост SMTP-сервера.\nПримеры:\n smtp.gmail.com\n smtp.yandex.ru"                                                                                                                                  env:"SMTP_SERV"     yaml:"host"`
	Port     string        `comment:"Порт SMTP-сервера.\nЧасто используемые:\n 465 — SMTPS (SSL)\n 587 — STARTTLS"                                                                                                                    env:"SMTP_PORT"     yaml:"port"`
	Email    string        `comment:"Email-адрес учетной записи,\nот имени которой будут отправляться письма.\nЭтот адрес используется в поле From."                                                                                  env:"SMTP_EMAIL"    yaml:"email"`
	Password secret.String `comment:"Пароль от email-учетной записи.\nОбычно это пароль приложения, а не основной пароль аккаунта."                                                                                                   env:"SMTP_PASSWORD" yaml:"password"`
}
