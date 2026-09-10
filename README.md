# RustDesk Support Setup

[![RustDesk](https://img.shields.io/badge/RustDesk-OSS-024EFF?logo=rustdesk&logoColor=white)](https://rustdesk.com/)
[![Go](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Windows](https://img.shields.io/badge/Client-Windows-0078D4?logo=windows&logoColor=white)](https://www.microsoft.com/windows/)
[![Docker](https://img.shields.io/badge/Server-Docker%20Compose-2496ED?logo=docker&logoColor=white)](https://docs.docker.com/compose/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

[English](README_EN.md) · **Русский**

Небольшая утилита для Windows, которая автоматически настраивает существующий клиент **RustDesk OSS** для работы с собственным self-hosted сервером.

Не требует RustDesk Pro, API или прав администратора.

Проект закрывает весь минимальный сценарий:

```text
VPS
 |
 +--> hbbs  ID / rendezvous
 |
 +--> hbbr  relay
 |
 +--> id_ed25519.pub
          |
          v
     support.exe
          |
          v
   RustDesk клиента
```

## Возможности

- настройка существующего RustDesk одним запуском `support.exe`
- собственный RustDesk OSS Server без RustDesk Pro
- `hbbs` и `hbbr` через Docker Compose
- постоянное хранение серверной пары ключей
- автоматическая настройка:
  - `custom-rendezvous-server`
  - `relay-server`
  - `key`
- сохранение остальных настроек RustDesk и истории подключений
- автоматический перезапуск установленного RustDesk
- relay по тому же домену, что и ID Server
- готовая схема обновления и backup сервера

## Что изменяет `support.exe`

Программа изменяет только три параметра в:

```text
%APPDATA%\RustDesk\config\RustDesk2.toml
```

```text
custom-rendezvous-server
relay-server
key
```

Остальная конфигурация файла сохраняется.

RustDesk должен быть установлен по стандартному пользовательскому пути:

```text
%LOCALAPPDATA%\rustdesk\rustdesk.exe
```

## Требования

### Сервер

```text
Linux:        Debian / Ubuntu
Public IPv4:  требуется
Domain name:  требуется
Docker:       Docker Engine + Docker Compose
```

Пример домена в инструкции:

```text
example.relay.net
```

DNS A-запись должна указывать на публичный IPv4 сервера.

Проверка:

```bash
getent ahostsv4 example.relay.net
```

### Порты

Минимальные порты RustDesk OSS Server:

| Port | Protocol | Purpose |
|---|---|---|
| `21115` | TCP | NAT type test |
| `21116` | TCP | ID / rendezvous / hole punching |
| `21116` | UDP | ID registration / heartbeat |
| `21117` | TCP | Relay |

`21118/tcp` и `21119/tcp` используются Web Client и для обычного RustDesk-клиента не требуются.

### Машина для сборки

```text
OS: Windows
Go: установлен
Git: установлен
```

На компьютере конечного пользователя Go и Git не нужны.

## Установка RustDesk OSS Server

Создайте DNS A-запись:

```text
example.relay.net -> PUBLIC_VPS_IP
```

Замените `example.relay.net` на свой домен и выполните на VPS от `root`:

```bash
set -euo pipefail

DOMAIN="example.relay.net"

# ------------------------------------------------------------
# Проверка
# ------------------------------------------------------------

if [ "$(id -u)" -ne 0 ]; then
    echo "ERROR: run as root"
    exit 1
fi

echo "DNS:"
getent ahostsv4 "$DOMAIN" || {
    echo "ERROR: $DOMAIN does not resolve"
    exit 1
}

# ------------------------------------------------------------
# Docker
# ------------------------------------------------------------

apt update
apt install -y ca-certificates curl

if ! command -v docker >/dev/null 2>&1; then
    curl -fsSL https://get.docker.com | sh
fi

docker --version
docker compose version

# ------------------------------------------------------------
# RustDesk
# ------------------------------------------------------------

mkdir -p /opt/rustdesk/data
cd /opt/rustdesk

cat >/opt/rustdesk/compose.yml <<'EOF'
services:
  hbbs:
    container_name: hbbs
    image: rustdesk/rustdesk-server:latest
    command: hbbs
    volumes:
      - ./data:/root
    network_mode: "host"
    restart: unless-stopped

  hbbr:
    container_name: hbbr
    image: rustdesk/rustdesk-server:latest
    command: hbbr
    volumes:
      - ./data:/root
    network_mode: "host"
    restart: unless-stopped
EOF

docker compose pull
docker compose up -d

# ------------------------------------------------------------
# Firewall
# ------------------------------------------------------------

if command -v ufw >/dev/null 2>&1 && ufw status | grep -q '^Status: active'; then
    ufw allow 21115/tcp
    ufw allow 21116/tcp
    ufw allow 21116/udp
    ufw allow 21117/tcp
else
    echo
    echo "NOTE: UFW is inactive or not installed."
    echo "Open manually:"
    echo "  21115/tcp"
    echo "  21116/tcp"
    echo "  21116/udp"
    echo "  21117/tcp"
fi

# ------------------------------------------------------------
# Ожидание генерации ключа hbbs
# ------------------------------------------------------------

for _ in $(seq 1 30); do
    [ -s /opt/rustdesk/data/id_ed25519.pub ] && break
    sleep 1
done

if [ ! -s /opt/rustdesk/data/id_ed25519.pub ]; then
    echo "ERROR: hbbs public key was not created"
    docker logs hbbs --tail 100
    exit 1
fi

# ------------------------------------------------------------
# Проверка
# ------------------------------------------------------------

echo
docker compose ps

echo
echo "Listening ports:"
ss -lntup | grep -E ':2111[5-9]\b' || true

echo
echo "============================================================"
echo "RustDesk Server is ready"
echo "============================================================"
echo
echo "ID Server:"
echo "  $DOMAIN"
echo
echo "Relay Server:"
echo "  $DOMAIN"
echo
echo "Public Key:"
cat /opt/rustdesk/data/id_ed25519.pub
echo
echo "============================================================"
```

При первом запуске `hbbs` автоматически создаёт серверную пару ключей:

```text
/opt/rustdesk/data/id_ed25519
/opt/rustdesk/data/id_ed25519.pub
```

Публичный ключ:

```bash
cat /opt/rustdesk/data/id_ed25519.pub
```

Приватный ключ:

```text
/opt/rustdesk/data/id_ed25519
```

Приватный ключ нельзя передавать клиентам или публиковать.

## Проверка сервера

Контейнеры:

```bash
cd /opt/rustdesk
docker compose ps
```

Логи `hbbs`:

```bash
docker logs hbbs --tail 100
```

Логи relay:

```bash
docker logs hbbr --tail 100
```

Порты:

```bash
ss -lntup | grep -E ':2111[5-9]\b'
```

Ожидаемый минимум:

```text
21115/tcp
21116/tcp
21116/udp
21117/tcp
```

## Сборка `support.exe`

Клонируйте репозиторий на Windows-машину с установленным Go:

```powershell
git clone https://github.com/dimadr/rustdesk-bootstrap.git
cd rustdesk-bootstrap
```

Получите публичный ключ на сервере:

```bash
cat /opt/rustdesk/data/id_ed25519.pub
```

Соберите `support.exe`:

```powershell
go build -buildvcs=false -ldflags "-H windowsgui -s -w -X main.serverValue=example.relay.net -X main.relayValue=example.relay.net -X main.keyValue=PUBLIC_KEY" -o support.exe .
```

Замените:

```text
example.relay.net -> ваш домен
PUBLIC_KEY        -> содержимое id_ed25519.pub
```

Если `main.relayValue` не задан, программа автоматически использует `main.serverValue` и как Relay Server.

Готовые бинарные файлы в репозитории не публикуются.

## Использование

На удалённом компьютере уже должен быть установлен RustDesk.

Запустите:

```text
support.exe
```

Программа:

```text
находит RustDesk
      |
      v
находит RustDesk2.toml
      |
      v
меняет только server / relay / key
      |
      v
останавливает запущенный RustDesk
      |
      v
запускает RustDesk снова
```

После этого клиент использует ваш self-hosted RustDesk Server.

## Архитектура

```text
                         Internet
                            |
             +--------------+--------------+
             |                             |
             v                             v
        hbbs :21116                   hbbr :21117
      ID / rendezvous                    relay
             |                             |
             +--------------+--------------+
                            |
                            v
                  RustDesk clients
```

`hbbs` используется для регистрации клиентов, rendezvous и попытки прямого соединения.

Если прямое соединение невозможно, трафик проходит через `hbbr`.

## Серверные файлы

```text
/opt/rustdesk/
├── compose.yml
└── data/
    ├── id_ed25519
    └── id_ed25519.pub
```

Каталог `data` подключается в оба контейнера:

```text
./data -> /root
```

Поэтому удаление или пересоздание контейнеров не меняет серверный ключ, пока сохранён `/opt/rustdesk/data`.

## Backup

Создать backup:

```bash
tar czf /root/rustdesk-data-backup.tar.gz -C /opt/rustdesk data
```

Проверить:

```bash
tar tzf /root/rustdesk-data-backup.tar.gz
```

Критически важный файл:

```text
data/id_ed25519
```

Backup рекомендуется хранить вне VPS.

## Обновление

Перед обновлением создайте backup:

```bash
tar czf /root/rustdesk-data-backup.tar.gz -C /opt/rustdesk data
```

Обновление:

```bash
cd /opt/rustdesk

docker compose pull
docker compose up -d

docker compose ps
```

Проверка:

```bash
docker logs hbbs --tail 50
docker logs hbbr --tail 50
```

Пара ключей сохраняется в `/opt/rustdesk/data` и при обычном обновлении не меняется.

## Security

Публиковать можно:

```text
домен ID Server
домен Relay Server
id_ed25519.pub
support.exe
```

Не публиковать:

```text
/opt/rustdesk/data/id_ed25519
backup-архивы с приватным ключом
доступ к VPS
SSH private keys
```

Для обычного RustDesk OSS Client не открывайте без необходимости:

```text
21118/tcp
21119/tcp
```

Эти порты предназначены для Web Client.

## Troubleshooting

### `support.exe` не находит RustDesk

Ожидаемый путь:

```text
%LOCALAPPDATA%\rustdesk\rustdesk.exe
```

Проверьте:

```powershell
Test-Path "$env:LOCALAPPDATA\rustdesk\rustdesk.exe"
```

### `support.exe` не находит конфигурацию

Ожидаемый файл:

```text
%APPDATA%\RustDesk\config\RustDesk2.toml
```

Проверьте:

```powershell
Test-Path "$env:APPDATA\RustDesk\config\RustDesk2.toml"
```

Если RustDesk только что установлен, запустите его хотя бы один раз, чтобы конфигурация была создана.

### Не появился `id_ed25519.pub`

Проверьте `hbbs`:

```bash
docker ps
docker logs hbbs --tail 100
ls -la /opt/rustdesk/data/
```

### Клиент видит сервер, но соединение идёт плохо

Проверьте relay:

```bash
docker logs -f hbbr
```

И firewall:

```bash
ss -lntup | grep -E ':2111[5-9]\b'
```

Должны быть доступны:

```text
21115/tcp
21116/tcp
21116/udp
21117/tcp
```

### Проверка DNS

```bash
getent ahostsv4 example.relay.net
```

Адрес должен соответствовать публичному IPv4 VPS.

## Удаление сервера

Сначала создайте backup:

```bash
tar czf /root/rustdesk-data-backup.tar.gz -C /opt/rustdesk data
```

Остановите контейнеры:

```bash
cd /opt/rustdesk
docker compose down
```

Полное удаление данных:

```bash
rm -rf /opt/rustdesk
```

Последняя команда уничтожает серверную пару ключей. После этого ранее настроенные клиенты потребуют новый публичный ключ.

## Ссылки

- [RustDesk](https://rustdesk.com/)
- [RustDesk Self-host Documentation](https://rustdesk.com/docs/en/self-host/)
- [RustDesk Server OSS Docker](https://rustdesk.com/docs/en/self-host/rustdesk-server-oss/docker/)
- [Docker Compose](https://docs.docker.com/compose/)
- [Go](https://go.dev/)

## Лицензия

MIT License — see [LICENSE](LICENSE)

## Disclaimer

Проект предоставляется как есть, без гарантий работы, совместимости или сохранности данных.

Администратор самостоятельно отвечает за сервер, DNS, firewall, обновления, резервные копии, безопасность инфраструктуры и соответствие применимому законодательству.

Перед использованием на важной системе проверьте доступность сервера, создайте backup приватного ключа и протестируйте подключение с внешней сети.
