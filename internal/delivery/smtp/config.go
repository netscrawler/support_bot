package smtp

type SMTPConfig struct {
	Host     string `env:"SMTP_SERV"     yaml:"host"`
	Port     string `env:"SMTP_PORT"     yaml:"port"`
	Email    string `env:"SMTP_EMAIL"    yaml:"email"`
	Password string `env:"SMTP_PASSWORD" yaml:"password"`
}
