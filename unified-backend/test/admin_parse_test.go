package test

import (
	"fmt"
	"testing"
)

// parsePositiveInt — локальная копия handler.parsePositiveInt для unit-тестов.
func parsePositiveInt(s string) (int, error) {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid character %q in %q", ch, s)
		}
		n = n*10 + int(ch-'0')
	}
	if n == 0 {
		return 0, fmt.Errorf("value must be positive")
	}
	return n, nil
}

func TestParsePositiveInt_Valid(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"1", 1},
		{"10", 10},
		{"100", 100},
		{"9999", 9999},
	}
	for _, tc := range cases {
		got, err := parsePositiveInt(tc.input)
		if err != nil {
			t.Errorf("input=%q: unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("input=%q: want %d, got %d", tc.input, tc.want, got)
		}
	}
}

func TestParsePositiveInt_Invalid(t *testing.T) {
	cases := []string{
		"",      // пустая строка
		"0",     // ноль — не positive
		"-1",    // отрицательное
		"abc",   // не число
		"1.5",   // float
		"1 2",   // пробел
		"12abc", // смешанный
	}
	for _, tc := range cases {
		_, err := parsePositiveInt(tc)
		if err == nil {
			t.Errorf("input=%q: expected error, got nil", tc)
		}
	}
}

func TestParsePositiveInt_Boundary(t *testing.T) {
	got, err := parsePositiveInt("200")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 200 {
		t.Errorf("want 200, got %d", got)
	}
}
