package smtp

type SMTPConfig struct {
	Host     string `env:"SMTP_SERV"     yaml:"host"     comment:"Хост SMTP-сервера.\nПримеры:\n smtp.gmail.com\n smtp.yandex.ru"`
	Port     string `env:"SMTP_PORT"     yaml:"port"     comment:"Порт SMTP-сервера.\nЧасто используемые:\n 465 — SMTPS (SSL)\n 587 — STARTTLS"`
	Email    string `env:"SMTP_EMAIL"    yaml:"email"    comment:"Email-адрес учетной записи,\nот имени которой будут отправляться письма.\nЭтот адрес используется в поле From."`
	Password string `env:"SMTP_PASSWORD" yaml:"password" comment:"\nПароль от email-учетной записи.\nОбычно это пароль приложения, а не основной пароль аккаунта."`
}
