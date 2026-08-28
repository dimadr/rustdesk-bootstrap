# RustDesk Support Setup

Небольшая утилита для Windows, автоматически настраивающая существующий клиент RustDesk OSS для работы с собственным (self-hosted) сервером.

Не требует RustDesk Pro, API или прав администратора.

## Что изменяет

Программа изменяет только три параметра в файле:

```text
%APPDATA%\RustDesk\config\RustDesk2.toml
```

- `custom-rendezvous-server`
- `relay-server`
- `key`

Все остальные настройки пользователя и история подключений сохраняются.

## Сборка

Соберите программу, указав адрес rendezvous-сервера, relay-сервера и публичный ключ:

```powershell
go build -buildvcs=false -ldflags "-H windowsgui -s -w -X main.serverValue=ВАШ_RENDEZVOUS_СЕРВЕР -X main.relayValue=ВАШ_RELAY_СЕРВЕР -X main.keyValue=ВАШ_ПУБЛИЧНЫЙ_КЛЮЧ" -o support.exe .
```

Пример:

```powershell
go build -buildvcs=false -ldflags "-H windowsgui -s -w -X main.serverValue=example.com -X main.relayValue=relay.example.com -X main.keyValue=PUBLIC_KEY" -o support.exe .
```

Готовые бинарные файлы не публикуются.

Если `main.relayValue` не указан, программа использует значение `main.serverValue` и для relay-сервера.

## Использование

Запустите `support.exe`.

Программа автоматически:

- найдёт существующую конфигурацию RustDesk;
- обновит только параметры self-hosted сервера;
- сохранит остальные настройки пользователя;
- запустит RustDesk.

## Лицензия

MIT
