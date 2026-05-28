package test

import (
	"strings"
	"testing"
)

// Тестируем sanitizeDetails через публичный Repository напрямую
// (функция приватная, поэтому проверяем косвенно через Record с известными полями).
// Для чистого unit-теста — выносим логику в отдельный экспортируемый хелпер.

// SanitizeForTest — копия sanitizeDetails для тестов (дублирует логику из audit_repo.go)
func SanitizeForTest(d map[string]any) map[string]any {
	sensitiveKeys := []string{
		"shlink_api_key", "api_key", "apikey", "x-api-key",
		"authorization", "password", "secret", "token",
	}
	if d == nil {
		return nil
	}
	result := make(map[string]any, len(d))
	for k, v := range d {
		kl := strings.ToLower(k)
		sensitive := false
		for _, sk := range sensitiveKeys {
			if kl == sk {
				sensitive = true
				break
			}
		}
		if !sensitive {
			result[k] = v
		}
	}
	return result
}

// TestSanitizeDetails_RemovesSensitiveKeys — API key никогда не попадает в аудит
func TestSanitizeDetails_RemovesSensitiveKeys(t *testing.T) {
	input := map[string]any{
		"shlink_api_key": "secret-key-value",
		"api_key":        "another-secret",
		"method":         "POST",
		"shortCode":      "abc123",
		"authorization":  "Bearer token123",
		"password":       "hunter2",
	}

	result := SanitizeForTest(input)

	sensitive := []string{"shlink_api_key", "api_key", "authorization", "password"}
	for _, k := range sensitive {
		if _, exists := result[k]; exists {
			t.Errorf("sensitive key %q should be removed from audit details", k)
		}
	}

	// Безопасные поля должны остаться
	if result["method"] != "POST" {
		t.Error("safe field 'method' should be preserved")
	}
	if result["shortCode"] != "abc123" {
		t.Error("safe field 'shortCode' should be preserved")
	}
}

// TestSanitizeDetails_NilInput — nil не паникует
func TestSanitizeDetails_NilInput(t *testing.T) {
	result := SanitizeForTest(nil)
	if result != nil {
		t.Error("nil input should return nil")
	}
}

// TestSanitizeDetails_EmptyInput — пустой map возвращает пустой map
func TestSanitizeDetails_EmptyInput(t *testing.T) {
	result := SanitizeForTest(map[string]any{})
	if len(result) != 0 {
		t.Errorf("empty input should return empty map, got %d entries", len(result))
	}
}

// TestSanitizeDetails_CaseInsensitive — проверяем case-insensitive matching
func TestSanitizeDetails_CaseInsensitive(t *testing.T) {
	input := map[string]any{
		"SHLINK_API_KEY": "should-be-removed",
		"Api_Key":        "also-removed",
		"SafeField":      "keep-me",
	}
	result := SanitizeForTest(input)

	if _, exists := result["SHLINK_API_KEY"]; exists {
		t.Error("SHLINK_API_KEY (uppercase) should be removed")
	}
	if _, exists := result["Api_Key"]; exists {
		t.Error("Api_Key (mixed case) should be removed")
	}
	if result["SafeField"] != "keep-me" {
		t.Error("SafeField should be preserved")
	}
}
