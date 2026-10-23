# Support TgBot

<!-- TOC -->
* [Support TgBot](#support-tgbot)
  * [Что умеет сервис](#что-умеет-сервис)
  * [Основной пайплайн](#основной-пайплайн)
  * [Требования](#требования)
  * [CLI и Конфигурация](#cli-и-конфигурация)
    * [Конфигурация](#конфигурация)
    * [Запуск приложения](#запуск-приложения)
  * [Управление данными (CTL)](#управление-данными-ctl)
  * [Telegram-бот](#telegram-бот)
  * [Деплой](#деплой)
  * [Отчеты и Обработка](#отчеты-и-обработка)
    * [Источники (Queries)](#источники-queries)
    * [Процессоры (Pipeline)](#процессоры-pipeline)
    * [Шаблоны](#шаблоны)
  * [Live preview шаблонов](#live-preview-шаблонов)
  * [Структура проекта](#структура-проекта)
  * [Разработка](#разработка)
  * [Примечания](#примечания)
<!-- TOC -->

Support TgBot — сервис для автоматической генерации отчетов по данным из различных источников (Metabase, Jira, AppMetrica) и доставки результатов в Telegram, Max Messenger, email и SMB-шару.

Приложение предоставляет мощный CLI интерфейс для управления конфигурацией, отчетами и скриптами обработки данных.

## Что умеет сервис

- **Сбор данных**: Поддержка Metabase (по UUID карточек), Jira и AppMetrica.
- **Обработка данных**:
    - Использование CEL-выражений для условий отправки.
    - **Pipeline процессоры**: обработка данных с помощью **Lua-скриптов** или **SQL-запросов (DuckDB)**.
- **Планировщик**: Запуск задач по расписанию (cron) из PostgreSQL.
- **Форматы отчетов**: `text`, `html`, `png`, `pdf`, `csv`, `xlsx`.
- **Доставка**: Telegram, Max Messenger, SMTP (email) и SMB-шары.
- **Управление**:
    - Административное и пользовательское меню в Telegram.
    - CLI для экспорта/импорта отчетов и управления Lua-плагинами.
- **Надежность**: Сохранение состояния отправленных сообщений и возможность их удаления.

## Основной пайплайн

1. **Scheduler** читает активные записи из таблицы `crons`.
2. По cron-событию **EventCreator** находит связанные отчеты.
3. **Orchestrator** загружает описание отчета, получателей, шаблоны и форматы экспорта.
4. **Generator** собирает данные из источников, пропускает их через **Pipeline** (Lua/SQL), проверяет условия `evaluate.expr`, генерирует файлы и отправляет сообщение через **SenderProvider**.
5. Результаты отправки сохраняются в БД.

## Требования

- Go `1.25.6`.
- PostgreSQL (используется для хранения настроек, отчетов и логов).
- Telegram bot token от `@BotFather`.
- Доступы к источникам данных (Metabase, Jira, AppMetrica).
- `wkhtmltopdf` для генерации PDF.
- Docker и Docker Compose (опционально).

## CLI и Конфигурация

Основной бинарник — `sbot` (собирается из `cmd/bot`).

### Конфигурация

Порядок загрузки конфига:
1. флаг `--config` (для команды `run`);
2. переменная окружения `CONFIG_PATH`;
3. файл `./config.yaml`;
4. переменные окружения и `.env`.

**Управление конфигурацией через CLI:**
```bash
# Проверить существующий конфиг
./sbot config validate -config config.yaml

# Создать пример конфигурации (YAML или ENV)
./sbot config create -format yaml -out config.yaml
./sbot config create -format env -out .env

# Обновить/нормализовать структуру конфига
./sbot config update -config config.yaml
```

### Запуск приложения

```bash
# Основной запуск
./sbot run -config config.yaml

# Упрощенный запуск (совместимость)
./sbot -config config.yaml
```

## Управление данными (CTL)

Команда `ctl` позволяет управлять отчетами и скриптами в базе данных.

```bash
# Экспорт всех отчетов из БД в JSON-файлы
./sbot ctl export -config config.yaml -out ./exported_reports

# Проверка JSON-файлов отчетов перед загрузкой
./sbot ctl validate -report "reports/*.json"

# Загрузка отчетов из файлов в БД
./sbot ctl apply -config config.yaml -report "reports/*.json"

# Работа с Lua-скриптами (плагинами)
./sbot ctl script create -out plugin.lua
./sbot ctl script save -config config.yaml -script plugin.lua
```

## Telegram-бот

Бот поддерживает разделение ролей (admin, user) и следующие команды:

- `/start` — главное меню (доступные отчеты, ручной запуск).
- `/admin` — панель управления для администраторов.
- `/help` — справка по командам бота.
- `/add` — добавить групповой чат в базу (только в группах).
- `/info` — получить ID текущего чата и thread ID.
- `/register` — регистрация в системе.

## HTTP API

Публичный HTTP API отдаёт сгенерированные отчеты по публичной ссылке. Спецификация — в [`swagger.yaml`](swagger.yaml).

- `GET /api/v1/public/report/{report_id}` — скачать сгенерированный отчёт по публичному UUID.

Ссылки управляются через CLI:

```bash
# ссылка на неделю (без -ttl или с -ttl 0 — бессрочная)
sbot ctl link create -config config/local.yaml -report my_report -ttl 168h
sbot ctl link list   -config config/local.yaml [-report my_report]
sbot ctl link revoke -config config/local.yaml <uuid>
```

Истёкшая или отозванная ссылка отвечает `410 Gone`.

Настройки сервера — секция `http` в конфиге (`host`, `port`, таймауты, лимиты на размер тела/заголовков). Эндпоинт не
требует аутентификации — не используйте его для отчетов с чувствительными данными без дополнительной защиты на уровне
сети/прокси.

## Деплой

Релиз автоматизирован через Ansible — см. [`deploy/README.md`](deploy/README.md). Поддерживаются два способа запуска
на сервере поверх одного и того же бинарника:

```bash
# Systemd-сервис на сервере
ansible-playbook -i deploy/ansible/inventory.ini deploy/ansible/deploy-systemd.yml

# Docker-контейнер на сервере (docker compose)
ansible-playbook -i deploy/ansible/inventory.ini deploy/ansible/deploy-docker.yml
```

Продовый PostgreSQL должен быть развёрнут отдельно — деплой его не поднимает.

## Отчеты и Обработка

### Источники (Queries)
Отчет может собирать данные из нескольких источников. В CEL и шаблонах данные доступны по `title` запроса.

#### Период (`period_from` / `period_to`)
В значениях `params`, начинающихся с `=` (CEL), доступны переменные `period_from` и `period_to` (timestamp): начало периода включительно, конец не включительно. В условии отчёта (`evaluation`) они недоступны — только в `params`. `${name}` в `query` подставляет значение параметра.

| Период | `period_from` | `period_to` |
|--------|---------------|-------------|
| `today` | сегодня 00:00 | завтра 00:00 |
| `yesterday` | вчера 00:00 | сегодня 00:00 |
| `week` | 7 полных дней до сегодня: сегодня 00:00 − 7 дней | сегодня 00:00 |
| `month` | 1-е число прошлого календарного месяца 00:00 | 1-е число текущего месяца 00:00 |

Время считается в часовом поясе процесса.

Поле отчёта `default_period` (по умолчанию `today`) задаёт период для запусков по cron и кнопки «▶️ Запустить». В боте («📤 Получить», ЛК) для отчётов, где в `params` есть `period_`, сначала показывается выбор периода (Сегодня / Вчера / Неделя / Месяц); для остальных отчётов выбор не показывается.

Пример: PostgreSQL-запрос за вчерашний день.

```json
{
  "name": "orders_daily",
  "title": "Заказы за день",
  "default_period": "yesterday",
  "queries": [
    {
      "card_uuid": "main",
      "title": "orders",
      "type": "pg",
      "query": "select id, amount from orders where created_at >= '${from}' and created_at < '${to}'",
      "params": {"from": "=period_from", "to": "=period_to"}
    }
  ],
  "recipients": [
    {"name": "ops chat", "type": "tg", "chat": {"chat_id": -1001234567890, "type": "supergroup", "is_active": true, "ch_type": "tg"}}
  ],
  "exports": [{"format": "xlsx", "file_name": "orders.xlsx"}],
  "evaluation": "size(report[\"orders\"]) > 0",
  "active": true,
  "crons": [{"name": "daily 09:00", "cron": "0 9 * * *", "is_active": true, "event_type": 0}]
}
```

Миграцию `migrations/init.sql/010_report_default_period.sql` на существующей БД нужно накатить вручную (init-скрипты выполняются только при первом старте пустой БД).

### Процессоры (Pipeline)
После сбора данные могут быть обработаны:
- **SQL (DuckDB)**: Выполнение аналитических запросов над собранными данными.
- **Lua**: Гибкая обработка данных с помощью скриптов (поддерживаются плагины).

### Шаблоны
Используются стандартные `text/template` и `html/template` с функциями **Sprig** и кастомными фильтрами форматирования.

### Дедупликация (`dedup`)
Флаг `"dedup": true` в JSON отчёта (по умолчанию выключен) не даёт отчёту на частом cron слать одно и то же. Запуск по cron не отправляется, если данные совпадают с последним успешным запуском `cron` или `manual`. В «🧾 Запуски» такой запуск виден как `skipped` / `duplicate`.

- Сравниваются все листы датасета после pipeline, кроме `_meta`. Порядок ключей в строке не важен, порядок строк важен.
- Изменчивое значение (время генерации и т.п.) кладите в лист `_meta`, иначе каждый запуск будет «новым».
- Ручной запуск, пробный запуск админа и получение из ЛК отправляются всегда; успешный ручной запуск становится точкой сравнения.

Пример: алерты проверяются раз в 5 минут, сообщение приходит только когда набор горящих алертов изменился.

```json
{
  "name": "firing_alerts",
  "title": "Горящие алерты",
  "queries": [
    {"card_uuid": "ops", "title": "firing", "params": {"state": "Alerting"}, "type": "grafana_alerts", "query": "active"}
  ],
  "recipients": [
    {"name": "ops chat", "type": "tg", "chat": {"chat_id": -1001234567890, "type": "supergroup", "is_active": true, "ch_type": "tg"}}
  ],
  "exports": [{"format": "xlsx", "file_name": "alerts.xlsx"}],
  "evaluation": "size(report[\"firing\"]) > 0",
  "active": true,
  "dedup": true,
  "crons": [{"name": "every 5 min", "cron": "*/5 * * * *", "is_active": true, "event_type": 0}]
}
```

## Live preview шаблонов

Инструмент `cmd/live-server` позволяет верстать шаблоны в реальном времени.

```bash
go run ./cmd/live-server --config=./config.json --templates=./reports
```

## Структура проекта

```text
cmd/bot/                 основной сервис (sbot)
cmd/live-server/         предпросмотр шаблонов
internal/collector/      сборщики (Metabase, Jira, AppMetrica, Zabbix)
internal/processor/      обработка (Lua, SQL/DuckDB)
internal/delivery/       доставка (Telegram, Max, SMTP, SMB)
internal/generator/      генерация отчетов
internal/sheduler/       планировщик
internal/tg_bot/         логика Telegram-бота
internal/cli/            реализация команд CLI
migrations/              миграции БД
```

## Разработка

```bash
make build   # Сборка sbot
make test    # Запуск тестов
make clean   # Очистка
```

## Примечания

- В репозитории есть локальные файлы `config/local.yaml`, `config/local_docker.yaml`, `.env`, `reports/` и тестовые SQL-данные. Перед запуском проверьте, что секреты и адреса соответствуют вашему окружению.
- Основной бинарник называется `sbot`.
- PDF-экспорт в контейнере уже получает `wkhtmltopdf` из Dockerfile.
