package shlinkctl

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Runner — абстракция над запуском shlink CLI.
// Позволяет переключаться между docker exec и нативным вызовом без изменения бизнес-логики.
type Runner interface {
	// GenerateAPIKey вызывает `shlink api-key:generate --name <name>`
	// и возвращает сгенерированный ключ (UUID).
	GenerateAPIKey(ctx context.Context, name string) (string, error)
}

// DockerRunner — запускает shlink через `docker exec <container> shlink ...`
type DockerRunner struct {
	ContainerName string // например "shlink-api"
}

func NewDockerRunner(containerName string) *DockerRunner {
	return &DockerRunner{ContainerName: containerName}
}

func (r *DockerRunner) GenerateAPIKey(ctx context.Context, name string) (string, error) {
	cmd := exec.CommandContext(ctx,
		"docker", "exec", r.ContainerName,
		"shlink", "api-key:generate", "--name", name,
	)
	return parseKeyFromOutput(cmd)
}

// NativeRunner — запускает shlink напрямую (продовый режим, shlink в PATH).
type NativeRunner struct {
	ShlinkBin string // путь до бинаря, например "/usr/local/bin/shlink"
}

func NewNativeRunner(shlinkBin string) *NativeRunner {
	bin := shlinkBin
	if bin == "" {
		bin = "shlink"
	}
	return &NativeRunner{ShlinkBin: bin}
}

func (r *NativeRunner) GenerateAPIKey(ctx context.Context, name string) (string, error) {
	cmd := exec.CommandContext(ctx, r.ShlinkBin, "api-key:generate", "--name", name)
	return parseKeyFromOutput(cmd)
}

// parseKeyFromOutput запускает команду и извлекает UUID из stdout.
// shlink выводит строку вида:
//
//	Generated API key: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
func parseKeyFromOutput(cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("shlink cli error: %w; stderr: %s", err, stderr.String())
	}

	out := stdout.String()
	key := extractUUID(out)
	if key == "" {
		return "", fmt.Errorf("shlink cli: cannot parse api key from output: %q", out)
	}
	return key, nil
}

// extractUUID ищет первый UUID-like токен в строке.
func extractUUID(s string) string {
	for _, token := range strings.Fields(s) {
		token = strings.Trim(token, `"'.,`)
		if isUUID(token) {
			return token
		}
	}
	return ""
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
