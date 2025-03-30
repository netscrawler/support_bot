# Support Bot

## Описание

Support Bot - это телеграм-бот, написанный на Go с использованием библиотеки `telebot`. Бот предоставляет функционал для управления пользователями и чатами, а также отправки уведомлений.

## Функционал

- Управление пользователями (добавление, удаление, список пользователей)
- Управление чатами (добавление, удаление, список чатов)
- Отправка уведомлений пользователям и в чаты

## Конфигурация

Для запуска бота необходимо настроить конфигурационный файл. В проекте есть два примера конфигурационных файлов: `config/example.env` и `config/example.yaml`. Вы можете выбрать любой из них в зависимости от предпочтений.

### Пример конфигурации в формате `.env`

```env
DATABASE_PORT=5433
DATABASE_HOST=localhost
DATABASE_USER=your_database_user
DATABASE_PASSWORD=your_password
DATABASE_NAME=notification_bot
APP_DEBUG=false
APP_HOST=0.0.0.0
APP_PORT=8080
DATABASE_CONNECT_TIMEOUT=10
BOT_POLL_TIMEOUT=10
SHUTDOWN_TIMEOUT=5
TELEGRAM_TOKEN=your_telegram_token
```

### Пример конфигурации в формате `.yaml`

```yaml
database:
  port: 5432
  host: localhost
  user: your_user
  password: your_password
  name: notification_bot
app:
  debug: false
  host: 0.0.0.0
  port: 8080
timeout:
  database_connect: 10s
  bot_poll: 10s
  shutdown: 5s
bot:
  telegram_token: your_telegram_token
```

После заполнения конфигурационного файла, сохраните его и запустите бота. Бот автоматически загрузит конфигурацию из файла при запуске.

## Настройка бд
    - Перед запуском бота нужно применить скрипты из ./internal/database/migrations/init.sql
    - 001_create_tables.sql - создаст нужные таблицы
    - 002_add_admin.sql - нужно вставить данные для админа
    - бд только Postgresql

## Сборка и запуск с использованием Docker

1. Склонируйте репозиторий:
   ```sh
   git clone https://github.com/netscrawler/support_bot.git
   cd support_bot
   ```

2. Соберите Docker-образ:
   ```sh
   docker build -t support_bot .
   ```

3. Запустите контейнер:
   ```sh
   docker run --env-file=config/example.env -p 8080:8080 support_bot
   ```


1. Используйте команды для взаимодействия с ботом:
   - `/start` - Начало работы с ботом
   - `/admin` - Доступ к административным функциям (для администраторов)
   - `/register` - Регистрация пользователя
   - `/subscribe` - Подписка на уведомления (для чатов)

2. Управляйте пользователями и чатами через меню бота.

3. Отправляйте уведомления пользователям и в чаты через соответствующие команды в меню.
## Идеи при разработке
  - Я придерживался методологии DDD - Долго Дорого Дерьмово
  - KISS? - в скубиду это показывали
  - Я вообще в шоке как это нечто работает
  - i`m use Neovim for code this btw))
