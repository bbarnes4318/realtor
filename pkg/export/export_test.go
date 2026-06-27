package export

import (
	"testing"
)

func TestCleanPhoneNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+1 (555) 019-2834", "15550192834"},
		{"555.019.2834", "5550192834"},
		{"abc-123", "123"},
		{"", ""},
	}

	for _, test := range tests {
		result := cleanPhoneNumber(test.input)
		if result != test.expected {
			t.Errorf("cleanPhoneNumber(%q) = %q; expected %q", test.input, result, test.expected)
		}
	}
}

func TestDeduplicationFilters(t *testing.T) {
	seenPhones := make(map[string]bool)
	seenURLs := make(map[string]bool)

	// Sample data (already normalized as they would be in the database)
	agent1Phone := "+1 555-111-2222"
	agent1URL := "https://realtor.com/realestateagents/agent1"

	agent2Phone := "+1 555-111-2222" // Duplicate clean phone!
	agent2URL := "https://realtor.com/realestateagents/agent2"

	// Process Agent 1
	clean1 := cleanPhoneNumber(agent1Phone)
	if clean1 != "" {
		seenPhones[clean1] = true
	}
	if agent1URL != "" {
		seenURLs[agent1URL] = true
	}

	// Process Agent 2
	clean2 := cleanPhoneNumber(agent2Phone)
	if seenPhones[clean2] {
		// correctly identified duplicate phone!
	} else {
		t.Error("failed to identify duplicate phone number")
	}

	if seenURLs[agent2URL] {
		t.Error("erroneously identified agent2 URL as duplicate")
	}
}
