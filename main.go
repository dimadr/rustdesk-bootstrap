package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	serverValue = ""
	keyValue    = ""
)

var desiredConfig = map[string]string{
	"custom-rendezvous-server": serverValue,
	"relay-server":             serverValue,
	"key":                      keyValue,
}

var desiredOrder = []string{
	"custom-rendezvous-server",
	"relay-server",
	"key",
}

func main() {
	if err := run(); err != nil {
		messageBox("RustDesk support setup", err.Error(), 0x00000010)
		os.Exit(1)
	}
}

func run() error {
	if err := validateBuildConfig(); err != nil {
		return err
	}

	rustDeskPath, err := findRustDesk()
	if err != nil {
		return err
	}

	configPath, err := findConfig()
	if err != nil {
		return err
	}

	if err := updateConfig(configPath); err != nil {
		return fmt.Errorf("failed to update RustDesk config: %w", err)
	}

	stopExistingRustDesk(rustDeskPath)

	cmd := exec.Command(rustDeskPath)
	cmd.Dir = filepath.Dir(rustDeskPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start RustDesk: %w", err)
	}

	return nil
}

func validateBuildConfig() error {
	if serverValue == "" {
		return errors.New("server value is not set at build time")
	}
	if keyValue == "" {
		return errors.New("server public key is not set at build time")
	}
	return nil
}

func findRustDesk() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", errors.New("LOCALAPPDATA is not set")
	}

	path := filepath.Join(localAppData, "rustdesk", "rustdesk.exe")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("RustDesk was not found at %s", path)
		}
		return "", fmt.Errorf("failed to check RustDesk path: %w", err)
	}

	return path, nil
}

func findConfig() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", errors.New("APPDATA is not set")
	}

	path := filepath.Join(appData, "RustDesk", "config", "RustDesk2.toml")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("RustDesk config was not found at %s", path)
		}
		return "", fmt.Errorf("failed to check RustDesk config path: %w", err)
	}

	return path, nil
}

func updateConfig(path string) error {
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	updated, changed := rewriteToml(original)
	if !changed {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	return os.WriteFile(path, updated, info.Mode())
}

func rewriteToml(input []byte) ([]byte, bool) {
	newline := []byte("\n")
	if bytes.Contains(input, []byte("\r\n")) {
		newline = []byte("\r\n")
	}

	hadFinalNewline := len(input) > 0 && (input[len(input)-1] == '\n' || input[len(input)-1] == '\r')
	rawLines := splitLines(input)
	seen := make(map[string]bool, len(desiredConfig))
	changed := false

	for i, line := range rawLines {
		key, ok := tomlKey(line)
		if !ok {
			continue
		}
		value, wanted := desiredConfig[key]
		if !wanted {
			continue
		}

		replacement := []byte(fmt.Sprintf("%s = %q", key, value))
		if !bytes.Equal(line, replacement) {
			rawLines[i] = replacement
			changed = true
		}
		seen[key] = true
	}

	for _, key := range desiredOrder {
		if seen[key] {
			continue
		}
		rawLines = append(rawLines, []byte(fmt.Sprintf("%s = %q", key, desiredConfig[key])))
		changed = true
	}

	if !changed {
		return input, false
	}

	output := bytes.Join(rawLines, newline)
	if hadFinalNewline || len(input) == 0 {
		output = append(output, newline...)
	}

	return output, true
}

func splitLines(input []byte) [][]byte {
	trimmed := bytes.TrimSuffix(input, []byte("\n"))
	trimmed = bytes.TrimSuffix(trimmed, []byte("\r"))
	if len(trimmed) == 0 {
		return nil
	}

	normalized := bytes.ReplaceAll(trimmed, []byte("\r\n"), []byte("\n"))
	return bytes.Split(normalized, []byte("\n"))
}

func tomlKey(line []byte) (string, bool) {
	text := strings.TrimSpace(string(line))
	if text == "" || strings.HasPrefix(text, "#") || strings.HasPrefix(text, "[") {
		return "", false
	}

	idx := strings.IndexByte(text, '=')
	if idx < 0 {
		return "", false
	}

	key := strings.TrimSpace(text[:idx])
	key = strings.Trim(key, `"`)
	return key, key != ""
}

func stopExistingRustDesk(rustDeskPath string) {
	escapedPath := strings.ReplaceAll(rustDeskPath, "'", "''")
	script := fmt.Sprintf(
		"Get-CimInstance Win32_Process -Filter \"Name = 'rustdesk.exe'\" | Where-Object { $_.ExecutablePath -eq '%s' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }",
		escapedPath,
	)
	_ = exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script).Run()
}

func messageBox(title, text string, flags uintptr) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")

	titlePtr := utf16PtrFromString(title)
	textPtr := utf16PtrFromString(text)
	messageBoxW.Call(0, uintptr(unsafe.Pointer(textPtr)), uintptr(unsafe.Pointer(titlePtr)), flags)
}

func utf16PtrFromString(s string) *uint16 {
	encoded := utf16.Encode([]rune(s + "\x00"))
	return &encoded[0]
}
