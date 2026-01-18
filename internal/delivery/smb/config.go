package smb

type SMBConfig struct {
	Adress   string `env:"SMB_ADRESS"   yaml:"adress"`
	User     string `env:"SMB_USER"     yaml:"user"`
	Password string `env:"SMB_PASSWORD" yaml:"password"`
	Domain   string `env:"SMB_DOMAIN"   yaml:"domain"`
	Share    string `env:"SMB_SHARE"    yaml:"share"`
	Active   bool   `env:"SMB_ACTIVE"   yaml:"active"`
}
