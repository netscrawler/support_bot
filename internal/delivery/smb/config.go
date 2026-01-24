package smb

type SMBConfig struct {
	Address  string `env:"SMB_ADDRESS"  yaml:"address"  comment:"Адрес SMB-сервера в формате: host:port\nПример:192.168.1.10:445"`
	User     string `env:"SMB_USER"     yaml:"user"     comment:"Имя пользователя для аутентификации на SMB-сервере."`
	Password string `env:"SMB_PASSWORD" yaml:"password" comment:"Пароль пользователя для подключения к SMB-серверу."`
	Domain   string `env:"SMB_DOMAIN"   yaml:"domain"   comment:"Домен Windows / рабочая группа.\nОбычно: WORKGROUP.\nМожет быть пустым, если домен не используется."`
	Share    string `env:"SMB_SHARE"    yaml:"share"    comment:"Share — имя сетевой SMB-шары"`
	Active   bool   `env:"SMB_ACTIVE"   yaml:"active"   comment:"Active — включает использование SMB.\nЕсли false, подключение к SMB не выполняется"`
}
