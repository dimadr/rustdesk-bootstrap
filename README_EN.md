# RustDesk Support Setup

[![RustDesk](https://img.shields.io/badge/RustDesk-OSS-024EFF?logo=rustdesk&logoColor=white)](https://rustdesk.com/)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Windows](https://img.shields.io/badge/Client-Windows-0078D4?logo=windows&logoColor=white)](https://www.microsoft.com/windows/)
[![Docker](https://img.shields.io/badge/Server-Docker%20Compose-2496ED?logo=docker&logoColor=white)](https://docs.docker.com/compose/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**English** · [Русский](README.md)

A small Windows utility that automatically configures an existing **RustDesk OSS** client to use your own self-hosted server.

It does not require **RustDesk Pro**, an API, or administrator privileges.

The project covers the complete minimal workflow:

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
    RustDesk client
```

## Features

- configure an existing RustDesk installation by running `support.exe`
- use your own RustDesk OSS Server without RustDesk Pro
- deploy `hbbs` and `hbbr` with Docker Compose
- keep the server key pair persistent
- automatically configure:
  - `custom-rendezvous-server`
  - `relay-server`
  - `key`
- preserve all other RustDesk settings and connection history
- automatically restart the installed RustDesk client
- use the same domain for the ID Server and Relay Server
- documented server update and backup procedure

## What `support.exe` changes

The program changes only three parameters in:

```text
%APPDATA%\RustDesk\config\RustDesk2.toml
```

```text
custom-rendezvous-server
relay-server
key
```

The rest of the configuration file is preserved.

RustDesk must be installed in the standard per-user location:

```text
%LOCALAPPDATA%\rustdesk\rustdesk.exe
```

## Requirements

### Server

```text
Linux:        Debian / Ubuntu
Public IPv4:  required
Domain name:  required
Docker:       Docker Engine + Docker Compose
```

The example domain used in this guide is:

```text
example.relay.net
```

The DNS A record must point to the public IPv4 address of the server.

Check it with:

```bash
getent ahostsv4 example.relay.net
```

### Ports

Minimum ports required by RustDesk OSS Server:

| Port | Protocol | Purpose |
|---|---|---|
| `21115` | TCP | NAT type test |
| `21116` | TCP | ID / rendezvous / hole punching |
| `21116` | UDP | ID registration / heartbeat |
| `21117` | TCP | Relay |

`21118/tcp` and `21119/tcp` are used by the Web Client and are not required for the standard RustDesk client.

### Build machine

```text
OS: Windows
Go: 1.26
Git: installed
```

Go and Git are not required on the end user's computer.

## Installing RustDesk OSS Server

Create a DNS A record:

```text
example.relay.net -> PUBLIC_VPS_IP
```

Replace `example.relay.net` with your own domain and run the following block on the VPS as `root`:

```bash
set -euo pipefail

DOMAIN="example.relay.net"

# ------------------------------------------------------------
# Validation
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
# Wait for hbbs to generate its public key
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
# Verification
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

On the first start, `hbbs` automatically creates the server key pair:

```text
/opt/rustdesk/data/id_ed25519
/opt/rustdesk/data/id_ed25519.pub
```

Show the public key with:

```bash
cat /opt/rustdesk/data/id_ed25519.pub
```

The private key is:

```text
/opt/rustdesk/data/id_ed25519
```

Do not distribute or publish the private key.

## Server verification

Containers:

```bash
cd /opt/rustdesk
docker compose ps
```

`hbbs` logs:

```bash
docker logs hbbs --tail 100
```

Relay logs:

```bash
docker logs hbbr --tail 100
```

Listening ports:

```bash
ss -lntup | grep -E ':2111[5-9]\b'
```

Expected minimum:

```text
21115/tcp
21116/tcp
21116/udp
21117/tcp
```

## Building `support.exe`

Clone the repository on a Windows machine with Go installed:

```powershell
git clone https://github.com/dimadr/rustdesk-bootstrap.git
cd rustdesk-bootstrap
```

Get the server public key:

```bash
cat /opt/rustdesk/data/id_ed25519.pub
```

Build `support.exe`:

```powershell
go build -buildvcs=false -ldflags "-H windowsgui -s -w -X main.serverValue=example.relay.net -X main.relayValue=example.relay.net -X main.keyValue=PUBLIC_KEY" -o support.exe .
```

Replace:

```text
example.relay.net -> your domain
PUBLIC_KEY        -> contents of id_ed25519.pub
```

If `main.relayValue` is not set, the program automatically uses `main.serverValue` as the Relay Server as well.

Prebuilt binaries are not published in this repository.

## Usage

RustDesk must already be installed on the remote computer.

Run:

```text
support.exe
```

The program:

```text
finds RustDesk
      |
      v
finds RustDesk2.toml
      |
      v
changes only server / relay / key
      |
      v
stops the running RustDesk process
      |
      v
starts RustDesk again
```

The client will then use your self-hosted RustDesk Server.

## Architecture

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

`hbbs` handles client registration, rendezvous, and attempts to establish a direct connection.

If a direct connection cannot be established, traffic is relayed through `hbbr`.

## Server files

```text
/opt/rustdesk/
├── compose.yml
└── data/
    ├── id_ed25519
    └── id_ed25519.pub
```

The `data` directory is mounted into both containers:

```text
./data -> /root
```

Because of this, deleting or recreating the containers does not change the server key as long as `/opt/rustdesk/data` is preserved.

## Backup

Create a backup:

```bash
tar czf /root/rustdesk-data-backup.tar.gz -C /opt/rustdesk data
```

Verify it:

```bash
tar tzf /root/rustdesk-data-backup.tar.gz
```

The critical file is:

```text
data/id_ed25519
```

Important backup copies should be stored outside the VPS.

## Updating

Create a backup before updating:

```bash
tar czf /root/rustdesk-data-backup.tar.gz -C /opt/rustdesk data
```

Update the containers:

```bash
cd /opt/rustdesk

docker compose pull
docker compose up -d

docker compose ps
```

Verify:

```bash
docker logs hbbs --tail 50
docker logs hbbr --tail 50
```

The key pair is stored in `/opt/rustdesk/data` and does not change during a normal update.

## Security

Safe to publish or distribute:

```text
ID Server domain
Relay Server domain
id_ed25519.pub
support.exe
```

Do not publish:

```text
/opt/rustdesk/data/id_ed25519
backup archives containing the private key
VPS credentials
SSH private keys
```

For the standard RustDesk OSS Client, do not expose the following ports unless required:

```text
21118/tcp
21119/tcp
```

These ports are intended for the Web Client.

## Troubleshooting

### `support.exe` cannot find RustDesk

Expected path:

```text
%LOCALAPPDATA%\rustdesk\rustdesk.exe
```

Check it with:

```powershell
Test-Path "$env:LOCALAPPDATA\rustdesk\rustdesk.exe"
```

### `support.exe` cannot find the configuration file

Expected file:

```text
%APPDATA%\RustDesk\config\RustDesk2.toml
```

Check it with:

```powershell
Test-Path "$env:APPDATA\RustDesk\config\RustDesk2.toml"
```

If RustDesk has just been installed, start it at least once so that the configuration file is created.

### `id_ed25519.pub` was not created

Check `hbbs`:

```bash
docker ps
docker logs hbbs --tail 100
ls -la /opt/rustdesk/data/
```

### The client can see the server, but the connection is poor

Check the relay:

```bash
docker logs -f hbbr
```

Check the firewall and listening ports:

```bash
ss -lntup | grep -E ':2111[5-9]\b'
```

The following ports must be reachable:

```text
21115/tcp
21116/tcp
21116/udp
21117/tcp
```

### DNS verification

```bash
getent ahostsv4 example.relay.net
```

The returned address must match the public IPv4 address of the VPS.

## Removing the server

Create a backup first:

```bash
tar czf /root/rustdesk-data-backup.tar.gz -C /opt/rustdesk data
```

Stop the containers:

```bash
cd /opt/rustdesk
docker compose down
```

Remove all server data:

```bash
rm -rf /opt/rustdesk
```

The last command destroys the server key pair. Clients that were configured with the previous public key will require the new public key after the server is recreated.

## Links

- [RustDesk](https://rustdesk.com/)
- [RustDesk Self-host Documentation](https://rustdesk.com/docs/en/self-host/)
- [RustDesk Server OSS Docker](https://rustdesk.com/docs/en/self-host/rustdesk-server-oss/docker/)
- [Docker Compose](https://docs.docker.com/compose/)
- [Go](https://go.dev/)

## License

MIT License — see [LICENSE](LICENSE)

## Disclaimer

This project is provided as-is, without warranties of operation, compatibility, or data preservation.

The administrator is responsible for the server, DNS, firewall, updates, backups, infrastructure security, and compliance with applicable law.

Before using it on an important system, verify server connectivity, back up the private key, and test the connection from an external network.
