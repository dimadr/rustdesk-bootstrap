# RustDesk Support Setup

Small Windows utility that configures an existing RustDesk OSS client for a self-hosted server.

Works without RustDesk Pro, API or administrator privileges.

## What it changes

The utility updates only these keys in `%APPDATA%\RustDesk\config\RustDesk2.toml`:

- `custom-rendezvous-server`
- `relay-server`
- `key`

It does not replace the whole TOML file and does not delete existing user settings or connection history.

## Build

Build the executable with your own server name and public key:

```powershell
go build -buildvcs=false -ldflags "-H windowsgui -s -w -X main.serverValue=YOUR_SERVER -X main.keyValue=YOUR_SERVER_PUBLIC_KEY" -o support.exe .
```

Example:

```powershell
go build -buildvcs=false -ldflags "-H windowsgui -s -w -X main.serverValue=example.com -X main.keyValue=public-key-here" -o support.exe .
```

This project intentionally does not provide prebuilt binaries.

Build the executable with your own server name and public key.

## Usage

Run the generated `support.exe`.

The utility will:

- find the existing RustDesk configuration;
- update only the required self-hosted server settings;
- preserve all other settings;
- start RustDesk.

## License

MIT