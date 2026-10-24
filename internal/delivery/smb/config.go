package smb

import "support_bot/internal/pkg/secret"

type Config struct {
	Address  string        `comment:"Адрес SMB-сервера в формате: host:port\nПример:192.168.1.10:445"                                                                                                   env:"SMB_ADDRESS"  yaml:"address"`
	User     string        `comment:"Имя пользователя для аутентификации на SMB-сервере."                                                                                                               env:"SMB_USER"     yaml:"user"`
	Password secret.String `comment:"Пароль пользователя для подключения к SMB-серверу."                                                                                                                env:"SMB_PASSWORD" yaml:"password"`
	Domain   string        `comment:"Домен Windows / рабочая группа.\nОбычно: WORKGROUP.\nМожет быть пустым, если домен не используется."                                                               env:"SMB_DOMAIN"   yaml:"domain"`
	Share    string        `comment:"Share — имя сетевой SMB-шары"                                                                                                                                      env:"SMB_SHARE"    yaml:"share"`
	Active   bool          `comment:"Active — включает использование SMB.\nЕсли false, подключение к SMB не выполняется"                                                                                env:"SMB_ACTIVE"   yaml:"active"`
}
