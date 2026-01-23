# Support Bot

Telegram-бот для управления уведомлениями, пользователями и автоматизированной генерации отчетов.

## Описание

Support Bot - это телеграм-бот на Go с использованием библиотеки [telebot v4](https://github.com/tucnak/telebot), предоставляющий функционал для:
- Управления пользователями и чатами
- Отправки уведомлений и рассылок
- Автоматической генерации и отправки отчетов
- Интеграции с Metabase, SMB и SMTP

## Функциональность

### Основные возможности
- **Управление пользователями**: добавление, удаление, просмотр списка пользователей
- **Управление чатами**: добавление, удаление, просмотр списка чатов
- **Система уведомлений**: отправка уведомлений пользователям и в чаты
- **Генерация отчетов**: автоматическое создание отчетов из Metabase с расписанием (cron)
- **Интеграция с файловыми хранилищами**: поддержка SMB для работы с файлами
- **Email рассылка**: отправка отчетов через SMTP

### Команды бота
- `/start` - Начало работы с ботом
- `/admin` - Административная панель (только для администраторов)
- `/register` - Регистрация пользователя
- `/subscribe` - Подписка чата на уведомления

## Требования

- Go 1.25.6+
- PostgreSQL 12+
- Docker и Docker Compose (опционально)

## Быстрый старт

### Клонирование репозитория

```bash
git clone https://github.com/netscrawler/support_bot.git
cd support_bot
```

### Настройка базы данных

1. Создайте базу данных PostgreSQL
2. Примените миграции из директории `migrations/init.sql/` в следующем порядке:
   ```bash
   psql -U your_user -d notification_bot -f migrations/init.sql/001_create_tables.sql
   psql -U your_user -d notification_bot -f migrations/init.sql/002_create_report_tables.sql
   psql -U your_user -d notification_bot -f migrations/init.sql/003_add_admin.sql
   psql -U your_user -d notification_bot -f migrations/init.sql/004_insert_settings.sql
   ```

### Конфигурация

Создайте конфигурационный файл на основе примера:

```bash
cp config/config.example.yaml config/local.yaml
```

Отредактируйте `config/local.yaml`, указав необходимые параметры:

```yaml
log:
  level: info              # Уровень логирования: debug, info, warn, error
  file: ./log.log          # Путь к файлу логов
  output: [stdout, file]   # Куда выводить логи

metabase_domain: https://your-metabase-instance.com  # URL Metabase (если используется)

database:
  port: 5432
  host: localhost
  user: your_user
  password: your_password
  name: notification_bot

timeout:
  database_connect: 10s    # Таймаут подключения к БД
  bot_poll: 10s            # Таймаут опроса Telegram API
  shutdown: 5s             # Таймаут graceful shutdown

bot:
  telegram_token: your_telegram_bot_token
  CleanUpTime: 5m          # Время очистки старых сообщений

smb:                       # Настройки SMB (опционально)
  adress: //server/share
  user: username
  password: password
  domain: DOMAIN

smtp:                      # Настройки SMTP (опционально)
  host: smtp.example.com
  port: 587
  email: bot@example.com
  password: email_password
```

**Альтернативный способ**: Укажите путь к конфигу через переменную окружения:

```bash
export CONFIG_PATH=./config/local.yaml
```

## Запуск

### Локальный запуск

```bash
# Сборка
make build

# Запуск (автоматически соберет и запустит)
make run
```

**Примечание**: В `Makefile` можно изменить параметр `CONFIG_NAME` для указания нужного конфигурационного файла.

### Запуск через Docker Compose (рекомендуется)

```bash
# Запуск всех сервисов (бот, PostgreSQL, Samba)
docker-compose up -d

# Просмотр логов
docker-compose logs -f bottst

# Остановка
docker-compose down
```

Docker Compose автоматически:
- Создаст и инициализирует базу данных PostgreSQL
- Применит миграции из `migrations/init.sql/`
- Запустит бота с конфигурацией из `config/local_docker.yaml`
- Настроит Samba сервер для тестирования (опционально)

### Запуск через Docker (standalone)

```bash
# Сборка образа
docker build -t support_bot .

# Запуск контейнера
docker run -d \
  --name support_bot \
  -e CONFIG_PATH=/bot/config/local.yaml \
  -v $(pwd)/config:/bot/config:ro \
  support_bot
```

## Структура проекта

```
support_bot/
├── cmd/bot/              # Точка входа приложения
├── internal/
│   ├── app/              # Логика приложения
│   ├── config/           # Загрузка конфигурации
│   ├── models/           # Модели данных
│   ├── postgres/         # Работа с БД
│   ├── tg_bot/           # Telegram bot handlers
│   ├── collector/        # Сбор данных
│   ├── evaluator/        # Обработка выражений
│   ├── exporter/         # Экспорт отчетов
│   ├── generator/        # Генерация отчетов
│   ├── orchestrator/     # Оркестрация процессов
│   └── sheduler/         # Планировщик задач
├── config/               # Конфигурационные файлы
├── migrations/           # SQL миграции
└── docker-compose.yaml   # Docker Compose конфигурация
```

## Разработка

### Сборка

```bash
make build
```

Параметры сборки включают:
- Версию из Git тега
- Commit hash
- Время сборки

### Очистка

```bash
make clean
```

## Лицензия

Проект распространяется "как есть" без каких-либо гарантий.
