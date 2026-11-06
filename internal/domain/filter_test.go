package domain

import (
	"testing"
)

func TestFilterMatching(t *testing.T) {
	tests := []struct {
		name     string
		filter   *Filter
		msgType  string
		event    string
		eventKey string
		expected bool
	}{
		{
			name:     "nil filter matches all",
			filter:   nil,
			msgType:  "event",
			event:    "CLICK",
			eventKey: "MENU_1",
			expected: true,
		},
		{
			name:     "empty filter matches all",
			filter:   &Filter{},
			msgType:  "text",
			event:    "",
			eventKey: "",
			expected: true,
		},
		{
			name:     "match msgType exact",
			filter:   &Filter{MsgTypes: []string{"text", "image"}},
			msgType:  "TEXT",
			event:    "",
			eventKey: "",
			expected: true,
		},
		{
			name:     "mismatch msgType",
			filter:   &Filter{MsgTypes: []string{"event"}},
			msgType:  "text",
			event:    "",
			eventKey: "",
			expected: false,
		},
		{
			name:     "match eventKeyPrefix",
			filter:   &Filter{EventKeyPrefixes: []string{"MENU_", "QR_"}},
			msgType:  "event",
			event:    "CLICK",
			eventKey: "MENU_USER_INFO",
			expected: true,
		},
		{
			name:     "mismatch eventKeyPrefix",
			filter:   &Filter{EventKeyPrefixes: []string{"MENU_"}},
			msgType:  "event",
			event:    "CLICK",
			eventKey: "ACTION_SUBMIT",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.filter.Matches(tc.msgType, tc.event, tc.eventKey)
			if got != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}
