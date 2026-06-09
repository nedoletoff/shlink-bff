package shlinkctl

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Runner — абстракция над запуском shlink CLI.
type Runner interface {
	// GenerateAPIKey вызывает `shlink api-key:generate --name <name>`
	// и возвращает сгенерированный ключ (UUID).
	GenerateAPIKey(ctx context.Context, name string) (string, error)

	// DeleteAPIKey вызывает `shlink api-key:delete --name <name>`.
	// Если ключа нет — не возвращает ошибку (idempotent).
	DeleteAPIKey(ctx context.Context, name string) error
}

// ─── DockerRunner ─────────────────────────────────────────────────────────────

type DockerRunner struct {
	ContainerName string
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

func (r *DockerRunner) DeleteAPIKey(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx,
		"docker", "exec", r.ContainerName,
		"shlink", "api-key:delete", "--name", name, "--no-interaction",
	)
	return runIgnoreNotFound(cmd)
}

// ─── NativeRunner ─────────────────────────────────────────────────────────────

type NativeRunner struct {
	ShlinkBin string
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

func (r *NativeRunner) DeleteAPIKey(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, r.ShlinkBin, "api-key:delete", "--name", name, "--no-interaction")
	return runIgnoreNotFound(cmd)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

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

// runIgnoreNotFound запускает команду и игнорирует ошибку «ключ не найден».
func runIgnoreNotFound(cmd *exec.Cmd) error {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		combined := stdout.String() + stderr.String()
		// shlink печатает что-то вроде "No API key was found" или "not found" — пропускаем
		if strings.Contains(strings.ToLower(combined), "not found") ||
			strings.Contains(strings.ToLower(combined), "no api key") {
			return nil
		}
		return fmt.Errorf("shlink cli error: %w; stderr: %s", err, stderr.String())
	}
	return nil
}

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
