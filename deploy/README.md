# Деплой sbot

Автоматизирует релиз, который раньше делался вручную через `task deploy`
(сборка → `scp` бинарника → ручной рестарт на сервере). Поддерживаются два
способа запуска на сервере — выберите один, оба используют один и тот же
бинарник, собранный локально через `task docker-run`.

## Подготовка (один раз)

1. Создайте `ansible/inventory.ini` из примера и заполните реальным адресом
   и пользователем сервера (файл гитигнорен, как и `vault.yml`):
   ```bash
   cp deploy/ansible/inventory.ini.example deploy/ansible/inventory.ini
   ```
2. Заполните несекретные поля в `ansible/group_vars/production.yml`
   (`metabase_domain`, `jira_host`, SMB/SMTP-адреса и т.д.).
3. Создайте секреты:
   ```bash
   cd deploy/ansible
   cp group_vars/production/vault.yml.example group_vars/production/vault.yml
   # заполните реальными паролями/токенами
   ansible-vault encrypt group_vars/production/vault.yml
   ```

## Systemd (бинарник напрямую на сервере)

```bash
cd deploy/ansible
ansible-playbook --ask-vault-pass deploy-systemd.yml
```

Каждый запуск: собирает `bin/sbot`, кладёт его в
`{{ release_path }}/releases/<timestamp>/`, переключает симлинк `current`,
устанавливает/обновляет `sbot.service` и перезапускает его. Хранится
последние `keep_releases` релизов — откат делается переключением симлинка
`current` на предыдущий каталог в `releases/` и `systemctl restart sbot`.

## Docker (контейнер на сервере)

```bash
cd deploy/ansible
ansible-playbook --ask-vault-pass deploy-docker.yml
```

Собирает `bin/sbot`, упаковывает его в образ по `deploy/Dockerfile`,
передаёт образ и `deploy/docker-compose.yaml` на сервер и поднимает через
`docker compose up -d`. Продовый PostgreSQL не разворачивается — ожидается,
что он уже доступен по адресу из `vault.yml`.

## Локальная проверка Docker-образа без Ansible

```bash
task docker-run
docker build -f deploy/Dockerfile -t sbot:local .
SBOT_IMAGE_TAG=local docker compose -f deploy/docker-compose.yaml up
```

## Вне рамок

- Продовый PostgreSQL и применение SQL-миграций (`migrations/init.sql/`) —
  не автоматизированы, применяются отдельно до деплоя.
- Секреты — только через `ansible-vault`, ничего не коммитить в открытом виде.
