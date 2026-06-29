# Support Bot

<!-- TOC -->
* [Support Bot](#support-bot)
  * [Что умеет сервис](#что-умеет-сервис)
  * [Основной пайплайн](#основной-пайплайн)
  * [Требования](#требования)
  * [Конфигурация](#конфигурация)
  * [Запуск локально](#запуск-локально)
  * [Запуск через Docker Compose](#запуск-через-docker-compose)
  * [База данных](#база-данных)
  * [Telegram-бот](#telegram-бот)
  * [Отчеты](#отчеты)
  * [Live preview шаблонов](#live-preview-шаблонов)
  * [Структура проекта](#структура-проекта)
  * [Разработка](#разработка)
  * [Примечания](#примечания)
<!-- TOC -->

Support Bot — сервис для автоматической генерации отчетов по данным Metabase и доставки результатов в Telegram, email и SMB-шару.

Приложение состоит из двух CLI:

- `cmd/bot` — основной Telegram-бот и фоновый пайплайн отчетов.
- `cmd/live-server` — dev-сервер для предпросмотра HTML/text-шаблонов отчетов на данных из Metabase.

## Что умеет сервис

- Загружает расписания из PostgreSQL и запускает задачи через cron.
- Получает данные из Metabase по UUID карточек.
- Проверяет условия отправки через CEL-выражения.
- Формирует отчеты в форматах `text`, `html`, `png`, `pdf`, `csv`, `xlsx`.
- Отправляет результаты в Telegram-чаты, на email через SMTP и в SMB-шару.
- Позволяет администраторам управлять пользователями, чатами и расписаниями из Telegram.
- Позволяет пользователям запускать доступные отчеты вручную из Telegram.
- Сохраняет отправленные Telegram-сообщения и может удалять их по событию.

## Основной пайплайн

1. Scheduler читает активные записи из таблицы `crons`.
2. По cron-событию EventCreator находит связанные отчеты.
3. Orchestrator загружает описание отчета, получателей, шаблоны и форматы экспорта.
4. Generator собирает данные из Metabase, проверяет `evaluate.expr`, генерирует файлы и отправляет сообщение.
5. Результаты отправки Telegram сохраняются в `sent_messages`.

## Требования

- Go `1.25.6`.
- PostgreSQL. В `docker-compose.yaml` используется `postgres:9.6`.
- Telegram bot token от `@BotFather`.
- Доступ к Metabase.
- `wkhtmltopdf` для PDF-экспорта.
- Docker и Docker Compose, если запуск идет в контейнерах.

## Конфигурация

Конфиг загружается в таком порядке:

1. флаг `--config`;
2. переменная окружения `CONFIG_PATH`;
3. файл `./config.yaml`;
4. переменные окружения и `.env`, если файл конфигурации не найден.

Пример YAML находится в `config/config.example.yaml`, пример env-файла — в `config/example.env`.

```bash
cp config/config.example.yaml config/local.yaml
```

Минимально нужны:

- `metabase_domain` — адрес Metabase;
- `database` — подключение к PostgreSQL;
- `bot.telegram_token` — токен Telegram-бота;
- `smtp` — SMTP-доступ, если используются email-получатели;
- `smb` — SMB-доступ, если используются SMB-получатели.

SMB можно отключить через `smb.active: false`.

## Запуск локально

```bash
make build
make run
```

`make run` запускает `cmd/bot` с конфигом `config/local.yaml`. Имя файла можно поменять в `Makefile` через `CONFIG_NAME`.

Можно запустить напрямую:

```bash
go run ./cmd/bot --config=./config/local.yaml
```

Полезные флаги основного приложения:

```bash
go run ./cmd/bot -h
go run ./cmd/bot -v
go run ./cmd/bot -example-config
go run ./cmd/bot -example-env
```

## Запуск через Docker Compose

```bash
docker compose up -d
docker compose logs -f bottst
docker compose down
```

Compose поднимает:

- `bottst` — основной сервис;
- `db` — PostgreSQL с инициализацией из `migrations/init.sql`;
- `samba` — тестовая SMB-шара.

Контейнер бота использует `CONFIG_PATH=/sbot/config/local_docker.yaml`, поэтому перед запуском проверьте `config/local_docker.yaml`.

## База данных

Схема и начальные данные лежат в `migrations/init.sql`.

Ключевые сущности:

- `users` — Telegram-пользователи и роли `primary`, `admin`, `user`;
- `chats` — Telegram-чаты для доставки;
- `crons` — расписания;
- `reports` — отчеты;
- `queries` — Metabase-карточки;
- `evaluate` — CEL-условия отправки;
- `templates` — text/html-шаблоны;
- `export_formats` и `reports_export` — форматы и файлы экспорта;
- `recipients` и `reports_recipients` — получатели;
- `report_crons` — связь отчетов с расписаниями;
- `sent_messages` — сохраненные Telegram-сообщения.

Для локальной БД миграции можно применить вручную:

```bash
psql -U postgres -d bottst -f migrations/init.sql/001_create_tables.sql
psql -U postgres -d bottst -f migrations/init.sql/002_create_report_tables.sql
```

Остальные SQL-файлы в `migrations/init.sql` добавляют стартовые настройки, cron-задачи, шаблоны, получателей и конкретные отчеты.

## Telegram-бот

Поддерживаемые команды:

- `/register` — регистрация пользователя в личном чате с ботом.
- `/start` — пользовательское меню с ручным запуском отчетов.
- `/admin` — административное меню для пользователей с ролью администратора.
- `/info` — информация о групповом чате: title, chat id, thread id.
- `/add` — добавить текущий групповой чат в базу.
- `/sub` — добавить текущий групповой чат и сразу сделать его активным.

Админское меню позволяет:

- добавлять и удалять пользователей;
- выдавать роль `admin` или `user`;
- смотреть список пользователей;
- смотреть и удалять чаты;
- перезапускать и останавливать cron-рассылки;
- запускать отчеты вручную.

Команды `/info`, `/add` и `/sub` работают только в групповых чатах. `/start`, `/admin` и `/register` рассчитаны на личный чат с ботом.

## Отчеты

Отчет собирается из:

- одной или нескольких Metabase-карточек из `queries`;
- условия отправки из `evaluate`;
- одного или нескольких экспортов;
- списка получателей.

Поддерживаемые форматы:

- `text` — текст по Go `text/template`;
- `html` — HTML по Go `html/template`;
- `pdf` — PDF из HTML через `wkhtmltopdf`;
- `csv` — CSV-файлы по листам данных;
- `xlsx` — Excel-файл;
- `png` — PNG-рендер таблиц.

В шаблонах доступны функции Sprig и функции из `internal/pkg/text`: форматирование чисел, дат, строк, работа с map/list и вспомогательные функции для отчетов.

Условия отправки пишутся на CEL. Доступная переменная — `report`, где ключи верхнего уровня соответствуют `queries.title`.

Специальные условия:

- `[*]` — всегда отправлять;
- `[!*]` — никогда не отправлять.

Пример CEL:

```cel
size(report["sheet1"]) > 0
```

## Live preview шаблонов

`cmd/live-server` нужен для разработки шаблонов без запуска всего бота. Он собирает данные из Metabase, рендерит локальные `.html`, `.tmpl` и `.gotmpl` файлы и обновляет страницу при изменении шаблонов.

Сгенерировать пример конфига:

```bash
go run ./cmd/live-server -example-config > config.json
```

Запустить:

```bash
go run ./cmd/live-server \
  --config=./config.json \
  --templates=./reports
```

Сервер слушает `http://localhost:8080`.

Маршруты:

- `http://localhost:8080/html/<template>` — HTML-шаблон;
- `http://localhost:8080/text/<template>` — text-шаблон, обернутый в HTML для просмотра.

## Структура проекта

```text
cmd/bot/                 основной сервис
cmd/live-server/         предпросмотр шаблонов
internal/app/            сборка зависимостей приложения
internal/collector/      загрузка данных из Metabase
internal/evaluator/      CEL-условия отправки
internal/exporter/       экспортеры text/html/pdf/png/csv/xlsx
internal/generator/      генерация и отправка отчетов
internal/orchestrator/   маршрутизация событий к отчетам
internal/sheduler/       cron-планировщик
internal/tg_bot/         Telegram-роутер, меню, handlers, services
internal/delivery/       Telegram, SMTP и SMB-доставка
internal/postgres/       подключение к PostgreSQL
config/                  примеры и локальные конфиги
migrations/init.sql/     SQL-схема и стартовые данные
reports/                 локальные шаблоны для разработки
```

## Разработка

Сборка основного бота:

```bash
make build
```

Запуск тестов:

```bash
go test ./...
```

Очистка бинарника:

```bash
make clean
```

## Примечания

- В репозитории есть локальные файлы `config/local.yaml`, `config/local_docker.yaml`, `.env`, `reports/` и тестовые SQL-данные. Перед запуском проверьте, что секреты и адреса соответствуют вашему окружению.
- Основной бинарник называется `sbot`.
- PDF-экспорт в контейнере уже получает `wkhtmltopdf` из Dockerfile.
