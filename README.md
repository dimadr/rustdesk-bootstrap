# RustDesk Support Setup

Small Windows utility that configures an existing RustDesk OSS client for a self-hosted server.

It changes only these keys in `%APPDATA%\RustDesk\config\RustDesk2.toml`:

- `custom-rendezvous-server`
- `relay-server`
- `key`

The utility does not replace the whole TOML file and does not delete existing user settings or connection history.

## Build

Set your values at build time:

```powershell
go build -buildvcs=false -ldflags "-H windowsgui -s -w -X main.serverValue=YOUR_SERVER -X main.keyValue=YOUR_SERVER_PUBLIC_KEY" -o support.exe .
```

Example placeholders:

```powershell
go build -buildvcs=false -ldflags "-H windowsgui -s -w -X main.serverValue=example.com -X main.keyValue=public-key-here" -o support.exe .
```

Do not commit the built `support.exe` if it contains private deployment values.
