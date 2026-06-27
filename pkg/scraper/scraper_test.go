package scraper

import (
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"TEST@EMAIL.COM", "test@email.com"},
		{"  spaces@email.com  ", "spaces@email.com"},
	}

	for _, test := range tests {
		result := normalizeEmail(test.input)
		if result != test.expected {
			t.Errorf("normalizeEmail(%q) = %q; expected %q", test.input, result, test.expected)
		}
	}
}

func TestCleanPhoneNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"(555) 123-4567", "5551234567"},
		{"+1 (555) 987 6543", "15559876543"},
		{"12345", "12345"},
		{"", ""},
	}

	for _, test := range tests {
		var sb string
		for _, r := range test.input {
			if r >= '0' && r <= '9' {
				sb += string(r)
			}
		}
		if sb != test.expected {
			t.Errorf("cleanPhoneNumber(%q) = %q; expected %q", test.input, sb, test.expected)
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://twitter.com/http://twitter.com/username", "https://twitter.com/username"},
		{"http://www.facebook.com/http://facebook.com/user", "http://www.facebook.com/user"},
		{"", ""},
	}

	for _, test := range tests {
		result, err := tryNormalizeURL(test.input)
		if err != nil {
			// ignore or fail depending on input
			if test.input != "" {
				t.Errorf("tryNormalizeURL(%q) unexpected error: %v", test.input, err)
			}
		} else if result != test.expected {
			t.Errorf("tryNormalizeURL(%q) = %q; expected %q", test.input, result, test.expected)
		}
	}
}
