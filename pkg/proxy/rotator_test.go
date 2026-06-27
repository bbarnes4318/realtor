package proxy

import (
	"testing"
)

func TestParseProxyURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"http://user:pass@127.0.0.1:8080", "http://user:pass@127.0.0.1:8080", false},
		{"127.0.0.1:8989:user:pass", "http://user:pass@127.0.0.1:8989", false},
		{"127.0.0.1:8080", "http://127.0.0.1:8080", false},
		{"", "", true},
	}

	for _, tt := range tests {
		got, err := ParseProxyURL(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseProxyURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.expected {
			t.Errorf("ParseProxyURL(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
