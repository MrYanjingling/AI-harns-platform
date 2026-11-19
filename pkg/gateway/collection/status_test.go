package collection

import "testing"

func TestShouldStopBeforeConnect(t *testing.T) {
	cases := map[string]bool{
		CollectStatusCollecting:      true,
		CollectStatusCollectingError: true,
		CollectStatusUnconnected:     false,
		CollectStatusStopped:         false,
		CollectStatusEmptyVariable:   false,
		CollectStatusError:           false,
	}
	for status, expected := range cases {
		if actual := ShouldStopBeforeConnect(status); actual != expected {
			t.Fatalf("status %s: expected %v, got %v", status, expected, actual)
		}
	}
}
