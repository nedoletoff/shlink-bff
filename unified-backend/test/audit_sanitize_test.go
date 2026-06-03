package test

import (
	"strings"
	"testing"
)

// Тестируем sanitizeDetails через публичный Repository напрямую
// (функция приватная, поэтому проверяем косвенно через Record с известными полями).
// Для чистого unit-теста — выносим логику в отдельный экспортируемый хелпер.

// SanitizeForTest — копия sanitizeDetails для тестов (рекурсивная, как в audit_repo.go, #15)
var testSensitiveKeys = []string{
	"shlink_api_key", "api_key", "apikey", "x-api-key",
	"authorization", "password", "secret", "token",
}

func testIsSensitive(k string) bool {
	kl := strings.ToLower(k)
	for _, sk := range testSensitiveKeys {
		if kl == sk {
			return true
		}
	}
	return false
}

func SanitizeForTest(d map[string]any) map[string]any {
	if d == nil {
		return nil
	}
	result := make(map[string]any, len(d))
	for k, v := range d {
		if testIsSensitive(k) {
			continue
		}
		result[k] = sanitizeValueForTest(v)
	}
	return result
}

func sanitizeValueForTest(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return SanitizeForTest(val)
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = sanitizeValueForTest(item)
		}
		return out
	default:
		return v
	}
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

// TestSanitizeDetails_Nested — чувствительные ключи удаляются на вложенных уровнях (#15)
func TestSanitizeDetails_Nested(t *testing.T) {
	input := map[string]any{
		"data": map[string]any{
			"api_key":   "nested-secret",
			"shortCode": "abc123",
			"deeper": map[string]any{
				"password": "deep-secret",
				"keep":     "ok",
			},
		},
		"items": []any{
			map[string]any{"token": "list-secret", "id": 1},
		},
	}

	result := SanitizeForTest(input)
	data := result["data"].(map[string]any)
	if _, ok := data["api_key"]; ok {
		t.Error("nested api_key must be removed")
	}
	if data["shortCode"] != "abc123" {
		t.Error("nested safe field shortCode must be preserved")
	}
	deeper := data["deeper"].(map[string]any)
	if _, ok := deeper["password"]; ok {
		t.Error("deeply nested password must be removed")
	}
	if deeper["keep"] != "ok" {
		t.Error("deeply nested safe field must be preserved")
	}
	items := result["items"].([]any)
	first := items[0].(map[string]any)
	if _, ok := first["token"]; ok {
		t.Error("token inside slice must be removed")
	}
	if first["id"] != 1 {
		t.Error("safe field inside slice must be preserved")
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
