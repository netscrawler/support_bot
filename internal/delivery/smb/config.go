package smb

type SMBConfig struct {
	Address  string `env:"SMB_ADDRESS"  yaml:"address"`
	User     string `env:"SMB_USER"     yaml:"user"`
	Password string `env:"SMB_PASSWORD" yaml:"password"`
	Domain   string `env:"SMB_DOMAIN"   yaml:"domain"`
	Share    string `env:"SMB_SHARE"    yaml:"share"`
	Active   bool   `env:"SMB_ACTIVE"   yaml:"active"`
}
