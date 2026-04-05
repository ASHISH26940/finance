package controllers

import (
	"finance/models"
	"testing"
	"time"
)

func TestParsePageQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		fallback int
		want     int
	}{
		{name: "valid page", value: "3", fallback: 1, want: 3},
		{name: "empty uses fallback", value: "", fallback: 1, want: 1},
		{name: "zero uses fallback", value: "0", fallback: 2, want: 2},
		{name: "negative uses fallback", value: "-1", fallback: 4, want: 4},
		{name: "invalid uses fallback", value: "abc", fallback: 5, want: 5},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parsePageQuery(tt.value, tt.fallback)
			if got != tt.want {
				t.Fatalf("parsePageQuery(%q, %d) = %d, want %d", tt.value, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestParsePerPageQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		fallback int
		want     int
	}{
		{name: "valid per page", value: "25", fallback: 20, want: 25},
		{name: "empty uses fallback", value: "", fallback: 20, want: 20},
		{name: "invalid uses fallback", value: "abc", fallback: 10, want: 10},
		{name: "zero uses fallback", value: "0", fallback: 15, want: 15},
		{name: "caps at max", value: "500", fallback: 20, want: 100},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parsePerPageQuery(tt.value, tt.fallback)
			if got != tt.want {
				t.Fatalf("parsePerPageQuery(%q, %d) = %d, want %d", tt.value, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestCalculateTotalPages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		total   int64
		perPage int
		want    int
	}{
		{name: "no records", total: 0, perPage: 20, want: 0},
		{name: "exact division", total: 40, perPage: 20, want: 2},
		{name: "rounds up", total: 41, perPage: 20, want: 3},
		{name: "invalid per page", total: 10, perPage: 0, want: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := calculateTotalPages(tt.total, tt.perPage)
			if got != tt.want {
				t.Fatalf("calculateTotalPages(%d, %d) = %d, want %d", tt.total, tt.perPage, got, tt.want)
			}
		})
	}
}

func TestParseRecordTypeQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		want      models.RecordType
		wantError bool
	}{
		{name: "empty is allowed", value: "", want: ""},
		{name: "income", value: "income", want: models.Income},
		{name: "expense case insensitive", value: "ExPeNsE", want: models.Expense},
		{name: "invalid", value: "transfer", wantError: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseRecordTypeQuery(tt.value)
			if tt.wantError {
				if err == nil {
					t.Fatalf("parseRecordTypeQuery(%q) expected error", tt.value)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseRecordTypeQuery(%q) unexpected error: %v", tt.value, err)
			}

			if got != tt.want {
				t.Fatalf("parseRecordTypeQuery(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseOptionalUintQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		want      *uint64
		wantError bool
	}{
		{name: "empty is nil", value: "", want: nil},
		{name: "valid uint", value: "42", want: uint64Ptr(42)},
		{name: "zero rejected", value: "0", wantError: true},
		{name: "invalid rejected", value: "abc", wantError: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseOptionalUintQuery(tt.value)
			if tt.wantError {
				if err == nil {
					t.Fatalf("parseOptionalUintQuery(%q) expected error", tt.value)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseOptionalUintQuery(%q) unexpected error: %v", tt.value, err)
			}

			if !equalUint64Pointers(got, tt.want) {
				t.Fatalf("parseOptionalUintQuery(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseDateQuery(t *testing.T) {
	t.Parallel()

	t.Run("empty returns nil", func(t *testing.T) {
		t.Parallel()

		got, err := parseDateQuery("", false)
		if err != nil {
			t.Fatalf("parseDateQuery returned unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("parseDateQuery returned %v, want nil", got)
		}
	})

	t.Run("accepts rfc3339", func(t *testing.T) {
		t.Parallel()

		got, err := parseDateQuery("2026-04-05T10:00:00Z", false)
		if err != nil {
			t.Fatalf("parseDateQuery returned unexpected error: %v", err)
		}

		want := time.Date(2026, time.April, 5, 10, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("parseDateQuery returned %v, want %v", got, want)
		}
	})

	t.Run("accepts date only", func(t *testing.T) {
		t.Parallel()

		got, err := parseDateQuery("2026-04-05", false)
		if err != nil {
			t.Fatalf("parseDateQuery returned unexpected error: %v", err)
		}

		want := time.Date(2026, time.April, 5, 0, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("parseDateQuery returned %v, want %v", got, want)
		}
	})

	t.Run("date only end of day", func(t *testing.T) {
		t.Parallel()

		got, err := parseDateQuery("2026-04-05", true)
		if err != nil {
			t.Fatalf("parseDateQuery returned unexpected error: %v", err)
		}

		want := time.Date(2026, time.April, 5, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
		if !got.Equal(want) {
			t.Fatalf("parseDateQuery returned %v, want %v", got, want)
		}
	})

	t.Run("invalid date rejected", func(t *testing.T) {
		t.Parallel()

		if _, err := parseDateQuery("not-a-date", false); err == nil {
			t.Fatal("parseDateQuery expected error for invalid date")
		}
	})
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}

func equalUint64Pointers(a, b *uint64) bool {
	if a == nil || b == nil {
		return a == b
	}

	return *a == *b
}
